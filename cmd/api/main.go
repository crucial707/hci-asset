package main

import (
	"log"
	"net/http"
	"os"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/crucial707/hci-asset/internal/middleware"
)

func main() {

	// =====================
	// Load Config
	// =====================
	cfg := config.Load()

	// =====================
	// Database
	// =====================
	database, err := db.Connect(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBUser,
		cfg.DBPass,
	)
	if err != nil {
		log.Fatal(err)
	}

	// =====================
	// Repositories
	// =====================
	assetRepo := repo.NewAssetRepo(database)
	userRepo := repo.NewUserRepo(database)

	// =====================
	// Handlers
	// =====================
	assetHandler := &handlers.AssetHandler{
		Repo: assetRepo,
	}

	authHandler := &handlers.AuthHandler{
		UserRepo:  userRepo,
		JWTSecret: []byte(cfg.JWTSecret),
	}

	// =====================
	// Router
	// =====================
	r := chi.NewRouter()

	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)

	// =====================
	// Auth Routes
	// =====================
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)

	// =====================
	// Public Routes
	// =====================
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Get("/assets/list", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAsset)

	// =====================
	// Protected Routes (JWT)
	// =====================
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTMiddleware([]byte(cfg.JWTSecret))) // JWT required

		r.Post("/assets", assetHandler.CreateAsset)
		r.Put("/assets/{id}", assetHandler.UpdateAsset)
		r.Delete("/assets/{id}", assetHandler.DeleteAsset)
	})

	// =====================
	// Start Server
	// =====================
	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}

	log.Println("Server listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
