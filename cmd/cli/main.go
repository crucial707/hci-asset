package main

import (
	"database/sql"
	"log"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/crucial707/hci-asset/cmd/cli/assets"
	"github.com/crucial707/hci-asset/cmd/cli/auth"
	"github.com/crucial707/hci-asset/cmd/cli/scan"
	"github.com/crucial707/hci-asset/cmd/cli/users"

	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/repo"

	_ "github.com/lib/pq"
)

func main() {
	// ==========================
	// Setup DB Connection
	// ==========================
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "hci_asset")
	dbUser := getEnv("DB_USER", "postgres")
	dbPass := getEnv("DB_PASS", "password")

	connStr := "host=" + dbHost +
		" port=" + dbPort +
		" user=" + dbUser +
		" password=" + dbPass +
		" dbname=" + dbName +
		" sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	// ==========================
	// Initialize Repos
	// ==========================
	assetRepo := repo.NewAssetRepo(db)

	// ==========================
	// Initialize Handlers
	// ==========================
	assetHandler := &handlers.AssetHandler{Repo: assetRepo}
	// ==========================
	// Root CLI Command
	// ==========================
	rootCmd := &cobra.Command{
		Use:   "hci-asset",
		Short: "HCI Asset Management CLI",
		Long:  "Command-line interface for managing assets, users, and network scans.",
	}

	// ==========================
	// Attach CLI Modules
	// ==========================
	assets.InitAssets(rootCmd, assetHandler)
	users.InitUsers(rootCmd)
	scan.InitScan(rootCmd)
	auth.InitAuth(rootCmd)

	// ==========================
	// Execute CLI
	// ==========================
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing CLI: %v", err)
	}
}

// ==========================
// Helper: getEnv with default
// ==========================
func getEnv(key, fallback string) string {
	if val, ok := syscall.Getenv(key); ok {
		return val
	}
	return fallback
}
