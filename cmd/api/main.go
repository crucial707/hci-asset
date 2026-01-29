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
	db, err := sql.Open("postgres", "postgres://user:pass@localhost:5432/hci_assets?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	assetHandler := &handlers.AssetHandler{Repo: assetRepo}

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	r.Post("/assets", assetHandler.CreateAsset)
	r.Get("/assets", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAsset)
	r.Put("/assets/{id}", assetHandler.UpdateAsset)
	r.Delete("/assets/{id}", assetHandler.DeleteAsset)
	r.Post("/scan", assetHandler.ScanNetwork)
	r.Get("/scan/{id}", assetHandler.GetScanStatus)

	log.Println("API server running on :8080")
	http.ListenAndServe(":8080", r)
}
