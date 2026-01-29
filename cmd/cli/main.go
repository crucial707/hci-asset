package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/crucial707/hci-asset/cmd/cli/assets"
	"github.com/crucial707/hci-asset/cmd/cli/users"
)

func main() {

	// ==========================
	// Root Command
	// ==========================
	rootCmd := &cobra.Command{
		Use:   "hci-asset",
		Short: "HCI Asset Management CLI",
		Long:  "Command-line interface for managing assets, users, and network scans.",
	}

	// ==========================
	// Attach CLI Modules
	// ==========================
	assets.InitAssets(rootCmd)
	users.InitUsers(rootCmd)
	//	scan.InitScan(rootCmd)

	// ==========================
	// Execute CLI
	// ==========================
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("error executing CLI: %v", err)
	}
}
