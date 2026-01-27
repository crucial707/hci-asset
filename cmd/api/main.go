package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to DB
	database, err := db.Connect(
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBUser,
		cfg.DBPass,
	)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Successfully connected to the database")

	// Initialize repository
	assetRepo := repo.NewAssetRepo(database)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	// Create Asset endpoint
	http.HandleFunc("/assets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}

		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		asset, err := assetRepo.Create(input.Name, input.Description)
		if err != nil {
			http.Error(w, "failed to create asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(asset)

		http.HandleFunc("/assets/list", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			rows, err := database.Query("SELECT id, name, description, created_at FROM assets")
			if err != nil {
				http.Error(w, "failed to fetch assets", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var assets []models.Asset
			for rows.Next() {
				var a models.Asset
				if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.CreatedAt); err != nil {
					http.Error(w, "failed to scan row", http.StatusInternalServerError)
					return
				}
				assets = append(assets, a)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
		})

	})

	// Start server
	log.Println("Starting server on :" + cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
