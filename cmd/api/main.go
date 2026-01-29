package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

func main() {
	// ==========================
	// Database config
	// ==========================
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "assetdb")
	dbUser := getEnv("DB_USER", "assetuser")
	dbPass := getEnv("DB_PASS", "assetpass")

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to DB: %v", err)
	}
	defer db.Close()

	// ==========================
	// Initialize repos & handlers
	// ==========================
	assetRepo := repo.NewAssetRepo(db)
	assetHandler := &handlers.AssetHandler{Repo: assetRepo}

	// ==========================
	// Router setup
	// ==========================
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	r.Post("/assets", assetHandler.CreateAsset)
	r.Get("/assets", assetHandler.ListAssets)
	r.Get("/assets/{id}", assetHandler.GetAsset)
	r.Put("/assets/{id}", assetHandler.UpdateAsset)
	r.Delete("/assets/{id}", assetHandler.DeleteAsset)

	// Scan endpoints
	r.Post("/scan", assetHandler.ScanNetwork)
	r.Get("/scan/{id}", assetHandler.GetScanStatus)

	// ==========================
	// Start server
	// ==========================
	log.Println("API server running on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// ==========================
// Helper: getEnv with fallback
// ==========================
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
