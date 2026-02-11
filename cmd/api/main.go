package main

import (
	"database/sql"
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

	// Connect to Postgres (must match docker-compose: assetdb, assetuser, assetpass)
	db, err := sql.Open("postgres", "postgres://assetuser:assetpass@localhost:5432/assetdb?sslmode=disable")
	if err != nil {
		log.Fatal("DB open:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB ping failed (is Postgres running on localhost:5432?): ", err)
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

	// ==========================
	// Handlers
	// ==========================
	assetRepo := repo.NewAssetRepo(db)
	userRepo := repo.NewUserRepo(db)

	assetHandler := &handlers.AssetHandler{Repo: assetRepo}
	scanHandler := &handlers.ScanHandler{Repo: assetRepo}
	userHandler := &handlers.UserHandler{Repo: userRepo}
	authHandler := &handlers.AuthHandler{
		UserRepo: userRepo,
		Secret:   []byte(cfg.JWTSecret),
	}

	// ==========================
	// Router
	// ==========================
	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// ====== Auth endpoints (public) ======
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)

	// JWT middleware for protected routes
	jwtMiddleware := middleware.JWTMiddleware([]byte(cfg.JWTSecret))

	// ====== Asset endpoints (protected) ======
	r.With(jwtMiddleware).Post("/assets", assetHandler.CreateAsset)
	r.With(jwtMiddleware).Get("/assets", assetHandler.ListAssets)
	r.With(jwtMiddleware).Get("/assets/{id}", assetHandler.GetAsset)
	r.With(jwtMiddleware).Put("/assets/{id}", assetHandler.UpdateAsset)
	r.With(jwtMiddleware).Post("/assets/{id}/heartbeat", assetHandler.Heartbeat)
	r.With(jwtMiddleware).Delete("/assets/{id}", assetHandler.DeleteAsset)

	// ====== User endpoints (protected) ======
	r.With(jwtMiddleware).Post("/users", userHandler.CreateUser)
	r.With(jwtMiddleware).Get("/users", userHandler.ListUsers)
	r.With(jwtMiddleware).Get("/users/{id}", userHandler.GetUser)
	r.With(jwtMiddleware).Put("/users/{id}", userHandler.UpdateUser)
	r.With(jwtMiddleware).Delete("/users/{id}", userHandler.DeleteUser)

	// ====== Scan endpoints (protected, legacy paths) ======
	r.With(jwtMiddleware).Post("/scan", scanHandler.StartScan)
	r.With(jwtMiddleware).Get("/scan/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Post("/scan/{id}/cancel", scanHandler.CancelScan)

	// ====== Scan endpoints (protected, clean /scans API) ======
	r.With(jwtMiddleware).Post("/scans", scanHandler.StartScan)
	r.With(jwtMiddleware).Get("/scans/{id}", scanHandler.GetScanStatus)
	r.With(jwtMiddleware).Post("/scans/{id}/cancel", scanHandler.CancelScan)

	// ==========================
	log.Println("API server running on :8080")
	http.ListenAndServe(":8080", r)
}
