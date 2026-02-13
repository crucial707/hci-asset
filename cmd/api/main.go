package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/middleware"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/crucial707/hci-asset/internal/scheduler"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to Postgres (config from env: DB_HOST, DB_PORT, etc.; defaults match local docker-compose)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB open:", err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("DB ping failed (host=%s port=%s): %v", cfg.DBHost, cfg.DBPort, err)
	}

	// Run migrations from internal/db/migrations/ unless SKIP_MIGRATIONS is set
	if os.Getenv("SKIP_MIGRATIONS") == "" {
		if err := db.Run(dsn); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		log.Printf("migrations: up to date")
	}

	r, scanHandler, scheduleRepo := newRouter(dbConn, cfg)
	go scheduler.Run(scheduleRepo, func(target string) { scanHandler.StartScanTarget(target) })

	// ==========================
	addr := ":" + cfg.Port
	if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
		log.Printf("API server running with HTTPS on %s", addr)
		if err := http.ListenAndServeTLS(addr, cfg.TLSCertFile, cfg.TLSKeyFile, r); err != nil {
			log.Fatalf("ListenAndServeTLS failed: %v", err)
		}
	} else {
		log.Printf("API server running on %s (HTTP)", addr)
		if err := http.ListenAndServe(addr, r); err != nil {
			log.Fatalf("ListenAndServe failed: %v", err)
		}
	}
}

// newRouter builds the HTTP router with handlers and middleware (used by main and tests).
// Returns the router, ScanHandler (for scheduler), and ScheduleRepo (for scheduler).
func newRouter(db *sql.DB, cfg config.Config) (*chi.Mux, *handlers.ScanHandler, *repo.ScheduleRepo) {
	assetRepo := repo.NewAssetRepo(db)
	userRepo := repo.NewUserRepo(db)
	auditRepo := repo.NewAuditRepo(db)
	scheduleRepo := repo.NewScheduleRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)

	assetHandler := &handlers.AssetHandler{Repo: assetRepo, AuditRepo: auditRepo}
	scanHandler := &handlers.ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: cfg.NmapPath}
	userHandler := &handlers.UserHandler{Repo: userRepo, AuditRepo: auditRepo}
	auditHandler := &handlers.AuditHandler{Repo: auditRepo}
	scheduleHandler := &handlers.ScheduleHandler{Repo: scheduleRepo}
	authHandler := &handlers.AuthHandler{
		UserRepo: userRepo,
		Secret:   []byte(cfg.JWTSecret),
	}

	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Readiness: ping DB so orchestrators can fail unhealthy instances
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("db unreachable"))
			return
		}
		w.Write([]byte("ok"))
	})

	authLimiter := middleware.AuthRateLimiter()
	r.With(authLimiter.Middleware).Post("/auth/register", authHandler.Register)
	r.With(authLimiter.Middleware).Post("/auth/login", authHandler.Login)

	jwtMiddleware := middleware.JWTMiddleware([]byte(cfg.JWTSecret))
	adminOnly := middleware.RequireAdmin

	// Viewer (and admin): read-only
	r.With(jwtMiddleware).Get("/assets", assetHandler.ListAssets)
	r.With(jwtMiddleware).Get("/assets/{id}", assetHandler.GetAsset)
	r.With(jwtMiddleware).Get("/users", userHandler.ListUsers)
	r.With(jwtMiddleware).Get("/users/{id}", userHandler.GetUser)
	r.With(jwtMiddleware).Get("/audit", auditHandler.ListAudit)
	r.With(jwtMiddleware).Get("/scan/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Get("/scans", scanHandler.ListScans)
	r.With(jwtMiddleware).Get("/scans/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Get("/schedules", scheduleHandler.ListSchedules)
	r.With(jwtMiddleware).Get("/schedules/{id}", scheduleHandler.GetSchedule)

	// Admin only: create, update, delete, scan, heartbeat
	r.With(jwtMiddleware, adminOnly).Post("/assets", assetHandler.CreateAsset)
	r.With(jwtMiddleware, adminOnly).Put("/assets/{id}", assetHandler.UpdateAsset)
	r.With(jwtMiddleware, adminOnly).Post("/assets/{id}/heartbeat", assetHandler.Heartbeat)
	r.With(jwtMiddleware, adminOnly).Delete("/assets/{id}", assetHandler.DeleteAsset)
	r.With(jwtMiddleware, adminOnly).Post("/users", userHandler.CreateUser)
	r.With(jwtMiddleware, adminOnly).Put("/users/{id}", userHandler.UpdateUser)
	r.With(jwtMiddleware, adminOnly).Delete("/users/{id}", userHandler.DeleteUser)
	r.With(jwtMiddleware, adminOnly).Post("/scan", scanHandler.StartScan)
	r.With(jwtMiddleware, adminOnly).Post("/scan/{id}/cancel", scanHandler.CancelScan)
	r.With(jwtMiddleware, adminOnly).Post("/scans", scanHandler.StartScan)
	r.With(jwtMiddleware, adminOnly).Post("/scans/{id}/cancel", scanHandler.CancelScan)
	r.With(jwtMiddleware, adminOnly).Post("/schedules", scheduleHandler.CreateSchedule)
	r.With(jwtMiddleware, adminOnly).Put("/schedules/{id}", scheduleHandler.UpdateSchedule)
	r.With(jwtMiddleware, adminOnly).Delete("/schedules/{id}", scheduleHandler.DeleteSchedule)

	return r, scanHandler, scheduleRepo
}
