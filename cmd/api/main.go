package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/lib/pq"

	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

func main() {
	// Connect to Postgres
	db, err := sql.Open("postgres", "postgres://assetuser:assetpass@localhost:5432/assetdb?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// ==========================
	// Handlers
	// ==========================
	assetRepo := repo.NewAssetRepo(db)
	assetHandler := &handlers.AssetHandler{Repo: assetRepo}

	scanHandler := &handlers.ScanHandler{Repo: assetRepo}

	// ==========================
	// Router
	// ==========================
	r := chi.NewRouter()

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// ====== Asset endpoints ======
	r.Post("/assets", assetHandler.CreateAsset)
	r.Get("/assets", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAsset)
	r.Put("/assets/{id}", assetHandler.UpdateAsset)
	r.Delete("/assets/{id}", assetHandler.DeleteAsset)

	// ====== Scan endpoints ======
	r.Post("/scan", scanHandler.StartScan)
	r.Get("/scan/{id}", scanHandler.GetScanStatus)
	r.Post("/scan/{id}/cancel", scanHandler.CancelScan)

	// ==========================
	log.Println("API server running on :8080")
	http.ListenAndServe(":8080", r)
}
