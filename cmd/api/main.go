package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/lib/pq"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/middleware"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/crucial707/hci-asset/internal/scheduler"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

const defaultJWTSecret = "supersecretkey"

//go:embed openapi.json
var openAPISpec []byte

//go:embed docs.html
var swaggerUIHTML []byte

func main() {
	cfg := config.Load()

	// In non-dev mode, refuse to run with default or empty JWT_SECRET
	if cfg.Env != "dev" {
		if cfg.JWTSecret == "" || cfg.JWTSecret == defaultJWTSecret {
			log.Fatal("refusing to start: set JWT_SECRET to a secure value when ENV is not dev")
		}
	}

	// Structured logging: JSON in production when LOG_FORMAT=json
	if cfg.LogFormat == "json" {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB open:", err)
	}
	defer dbConn.Close()

	dbConn.SetMaxOpenConns(cfg.DBMaxOpenConns)
	dbConn.SetMaxIdleConns(cfg.DBMaxIdleConns)

	if err := dbConn.Ping(); err != nil {
		log.Fatalf("DB ping failed (host=%s port=%s): %v", cfg.DBHost, cfg.DBPort, err)
	}

	if os.Getenv("SKIP_MIGRATIONS") == "" {
		if err := db.Run(dsn); err != nil {
			log.Fatalf("migrations: %v", err)
		}
		slog.Info("migrations: up to date")
	}

	r, scanHandler, scheduleRepo := newRouter(dbConn, cfg)
	go scheduler.Run(scheduleRepo, func(target string) { scanHandler.StartScanTarget(context.Background(), target) })

	addr := ":" + cfg.Port
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			slog.Info("API server running with HTTPS", "addr", addr)
			if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("ListenAndServeTLS: %v", err)
			}
		} else {
			slog.Info("API server running", "addr", addr, "protocol", "HTTP")
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("ListenAndServe: %v", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
	slog.Info("API server stopped")
}

func serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(swaggerUIHTML)
}

// newRouter builds the HTTP router with handlers and middleware (used by main and tests).
// Returns the router, ScanHandler (for scheduler), and ScheduleRepo (for scheduler).
func newRouter(db *sql.DB, cfg config.Config) (*chi.Mux, *handlers.ScanHandler, *repo.ScheduleRepo) {
	assetRepo := repo.NewAssetRepo(db)
	userRepo := repo.NewUserRepo(db)
	auditRepo := repo.NewAuditRepo(db)
	scheduleRepo := repo.NewScheduleRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	savedScanRepo := repo.NewSavedScanRepo(db)

	assetHandler := &handlers.AssetHandler{Repo: assetRepo, AuditRepo: auditRepo}
	networkHandler := &handlers.NetworkHandler{Repo: assetRepo}
	scanHandler := &handlers.ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: cfg.NmapPath}
	savedScanHandler := &handlers.SavedScanHandler{Repo: savedScanRepo, ScanHandler: scanHandler}
	userHandler := &handlers.UserHandler{Repo: userRepo, AuditRepo: auditRepo}
	auditHandler := &handlers.AuditHandler{Repo: auditRepo}
	scheduleHandler := &handlers.ScheduleHandler{Repo: scheduleRepo}
	authHandler := &handlers.AuthHandler{
		UserRepo:    userRepo,
		Secret:      []byte(cfg.JWTSecret),
		ExpireHours: cfg.JWTExpireHours,
	}

	r := chi.NewRouter()
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(middleware.SecurityHeaders(cfg.TLSCertFile != "" && cfg.TLSKeyFile != ""))
	r.Use(middleware.MaxBytes(0)) // 0 => use default 1 MiB
	r.Use(middleware.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(middleware.RequestLog)
	r.Use(middleware.Prometheus)

	r.Handle("/metrics", promhttp.Handler())
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

	// OpenAPI spec and Swagger UI (no auth)
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(openAPISpec)
	})
	r.Get("/docs", serveSwaggerUI)

	// API v1 (versioned)
	r.Route("/v1", func(r chi.Router) {
		authLimiter := middleware.AuthRateLimiter()
		r.With(authLimiter.Middleware).Post("/auth/register", authHandler.Register)
		r.With(authLimiter.Middleware).Post("/auth/login", authHandler.Login)

		jwtMiddleware := middleware.JWTMiddleware([]byte(cfg.JWTSecret))
		adminOnly := middleware.RequireAdmin

		// Viewer (and admin): read-only
		r.With(jwtMiddleware).Get("/assets", assetHandler.ListAssets)
		r.With(jwtMiddleware).Get("/assets/{id}", assetHandler.GetAsset)
		r.With(jwtMiddleware).Get("/network/graph", networkHandler.NetworkGraph)
		r.With(jwtMiddleware).Get("/me", userHandler.Me)
		r.With(jwtMiddleware).Get("/users", userHandler.ListUsers)
		r.With(jwtMiddleware).Get("/users/{id}", userHandler.GetUser)
		r.With(jwtMiddleware).Get("/audit", auditHandler.ListAudit)
		r.With(jwtMiddleware).Get("/scan/{id}", scanHandler.GetScanStatus)
		r.With(jwtMiddleware).Get("/scans", scanHandler.ListScans)
		r.With(jwtMiddleware).Get("/scans/{id}", scanHandler.GetScanStatus)
		r.With(jwtMiddleware).Get("/saved-scans", savedScanHandler.ListSavedScans)
		r.With(jwtMiddleware).Get("/saved-scans/{id}", savedScanHandler.GetSavedScan)
		r.With(jwtMiddleware).Get("/schedules", scheduleHandler.ListSchedules)
		r.With(jwtMiddleware).Get("/schedules/{id}", scheduleHandler.GetSchedule)

		// Admin only: create, update, delete, scan, heartbeat
		r.With(jwtMiddleware, adminOnly).Post("/assets", assetHandler.CreateAsset)
		r.With(jwtMiddleware, adminOnly).Put("/assets/{id}", assetHandler.UpdateAsset)
		r.With(jwtMiddleware, adminOnly).Post("/assets/{id}/heartbeat", assetHandler.Heartbeat)
		r.With(jwtMiddleware, adminOnly).Delete("/assets/{id}", assetHandler.DeleteAsset)
		r.With(jwtMiddleware, adminOnly).Post("/users", userHandler.CreateUser)
		r.With(jwtMiddleware).Put("/users/{id}/password", userHandler.ChangePassword)
		r.With(jwtMiddleware, adminOnly).Put("/users/{id}", userHandler.UpdateUser)
		r.With(jwtMiddleware, adminOnly).Delete("/users/{id}", userHandler.DeleteUser)
		r.With(jwtMiddleware, adminOnly).Post("/scan", scanHandler.StartScan)
		r.With(jwtMiddleware, adminOnly).Post("/scan/{id}/cancel", scanHandler.CancelScan)
		r.With(jwtMiddleware, adminOnly).Post("/scans", scanHandler.StartScan)
		r.With(jwtMiddleware, adminOnly).Post("/scans/{id}/cancel", scanHandler.CancelScan)
		r.With(jwtMiddleware, adminOnly).Delete("/scans", scanHandler.ClearScans)
		r.With(jwtMiddleware, adminOnly).Post("/saved-scans", savedScanHandler.CreateSavedScan)
		r.With(jwtMiddleware, adminOnly).Put("/saved-scans/{id}", savedScanHandler.UpdateSavedScan)
		r.With(jwtMiddleware, adminOnly).Delete("/saved-scans/{id}", savedScanHandler.DeleteSavedScan)
		r.With(jwtMiddleware, adminOnly).Post("/saved-scans/{id}/run", savedScanHandler.RunSavedScan)
		r.With(jwtMiddleware, adminOnly).Post("/schedules", scheduleHandler.CreateSchedule)
		r.With(jwtMiddleware, adminOnly).Put("/schedules/{id}", scheduleHandler.UpdateSchedule)
		r.With(jwtMiddleware, adminOnly).Delete("/schedules/{id}", scheduleHandler.DeleteSchedule)
	})

	return r, scanHandler, scheduleRepo
}
