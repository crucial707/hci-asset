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
	"github.com/go-chi/chi/v5/middleware"
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

	// =====================
	// Handlers
	// =====================
	assetHandler := &handlers.AssetHandler{
		Repo:  assetRepo,
		Token: cfg.APIToken,
	}

	// =====================
	// Router
	// =====================
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// =====================
	// Routes
	// =====================

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Post("/assets", assetHandler.APITokenMiddleware(assetHandler.CreateAsset))
	r.Get("/assets/list", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAsset)
	r.Put("/assets/{id}", assetHandler.APITokenMiddleware(assetHandler.UpdateAsset))
	r.Delete("/assets/{id}", assetHandler.APITokenMiddleware(assetHandler.DeleteAsset))

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
