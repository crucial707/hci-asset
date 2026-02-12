package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/middleware"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to Postgres (config from env: DB_HOST, DB_PORT, etc.; defaults match local docker-compose)
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB open:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping failed (host=%s port=%s): %v", cfg.DBHost, cfg.DBPort, err)
	}

	// Ensure users table exists (migrations are not run automatically)
	if _, err := db.Exec("SELECT 1 FROM users LIMIT 0"); err != nil {
		log.Fatalf("users table missing. Create it in your Postgres (e.g. Docker container asset-postgres) with: "+
			"docker exec -i asset-postgres psql -U assetuser -d assetdb -c \"CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, username VARCHAR(255) NOT NULL UNIQUE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW());\" "+
			"Error: %v", err)
	}

	// Ensure assets table has last_seen column (idempotent; safe if column already exists or table missing)
	if _, err := db.Exec("ALTER TABLE assets ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ NULL"); err != nil {
		log.Printf("Warning: could not ensure last_seen on assets (table may not exist yet): %v", err)
	}

	r := newRouter(db, cfg)

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
func newRouter(db *sql.DB, cfg config.Config) *chi.Mux {
	assetRepo := repo.NewAssetRepo(db)
	userRepo := repo.NewUserRepo(db)

	assetHandler := &handlers.AssetHandler{Repo: assetRepo}
	scanHandler := &handlers.ScanHandler{Repo: assetRepo, NmapPath: cfg.NmapPath}
	userHandler := &handlers.UserHandler{Repo: userRepo}
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

	r.With(jwtMiddleware).Post("/assets", assetHandler.CreateAsset)
	r.With(jwtMiddleware).Get("/assets", assetHandler.ListAssets)
	r.With(jwtMiddleware).Get("/assets/{id}", assetHandler.GetAsset)
	r.With(jwtMiddleware).Put("/assets/{id}", assetHandler.UpdateAsset)
	r.With(jwtMiddleware).Post("/assets/{id}/heartbeat", assetHandler.Heartbeat)
	r.With(jwtMiddleware).Delete("/assets/{id}", assetHandler.DeleteAsset)

	r.With(jwtMiddleware).Post("/users", userHandler.CreateUser)
	r.With(jwtMiddleware).Get("/users", userHandler.ListUsers)
	r.With(jwtMiddleware).Get("/users/{id}", userHandler.GetUser)
	r.With(jwtMiddleware).Put("/users/{id}", userHandler.UpdateUser)
	r.With(jwtMiddleware).Delete("/users/{id}", userHandler.DeleteUser)

	r.With(jwtMiddleware).Post("/scan", scanHandler.StartScan)
	r.With(jwtMiddleware).Get("/scan/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Post("/scan/{id}/cancel", scanHandler.CancelScan)

	r.With(jwtMiddleware).Get("/scans", scanHandler.ListScans)
	r.With(jwtMiddleware).Post("/scans", scanHandler.StartScan)
	r.With(jwtMiddleware).Get("/scans/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Post("/scans/{id}/cancel", scanHandler.CancelScan)

	return r
}
