package main

// ========================
// IMPORTS
// ========================
import (
	"log"
	"net/http"

	"github.com/crucial707/hci-asset/internal/config"
	"github.com/crucial707/hci-asset/internal/db"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ========================
// MAIN ENTRY POINT
// ========================
func main() {

	// ========================
	// LOAD CONFIG
	// ========================
	cfg := config.Load()

	// ========================
	// CONNECT TO DATABASE
	// ========================
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

	// ========================
	// INITIALIZE REPOSITORIES & HANDLERS
	// ========================
	assetRepo := repo.NewAssetRepo(database)
	assetHandler := &handlers.AssetHandler{
		Repo:  assetRepo,
		Token: cfg.APIToken,
	}

	// ========================
	// SETUP ROUTER
	// ========================
	r := chi.NewRouter()

	// ---- HEALTH CHECK ----
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// ---- ASSET ROUTES ----
	r.Route("/assets", func(r chi.Router) {

		// POST /assets - CREATE
		r.Post("/", assetHandler.APITokenMiddleware(assetHandler.CreateAsset))

		// GET /assets/list - LIST ALL
		r.Get("/list", assetHandler.APITokenMiddleware(assetHandler.ListAssets))

		// GET /assets/{id} - GET SINGLE
		r.Get("/{id}", assetHandler.APITokenMiddleware(assetHandler.GetAsset))

		// DELETE /assets/{id} - DELETE
		r.Delete("/{id}", assetHandler.APITokenMiddleware(assetHandler.DeleteAsset))

		// PUT /assets/{id} - UPDATE
		r.Put("/{id}", assetHandler.APITokenMiddleware(assetHandler.UpdateAsset))
	})

	// ========================
	// START SERVER
	// ========================
	log.Println("Starting server on :" + cfg.Port)
	err = http.ListenAndServe(":"+cfg.Port, r)
	if err != nil {
		log.Fatal(err)
	}
}
