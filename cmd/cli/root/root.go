package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hci",
	Short: "HCI Asset Management CLI",
	Long: `Command line interface for interacting with the HCI Asset Management API.

You can use this CLI to:
  - List, create, update, and delete assets
  - Scan a network to discover devices automatically
  - Authenticate users if needed for protected operations`,
	SilenceUsage: true,
}

// Execute runs the root command
func Execute() {
	if err := GetRoot().Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// GetRoot returns the root Cobra command
func GetRoot() *cobra.Command {
	return rootCmd
}
