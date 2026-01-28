package main

import (
	"fmt"
	"os"

	"github.com/crucial707/hci-asset/cmd/cli/assets"
	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/crucial707/hci-asset/cmd/cli/scan"
	"github.com/crucial707/hci-asset/cmd/cli/users"
)

func main() {
	// Get the root command
	rootCmd := root.GetRoot()

	// Initialize CLI modules
	users.InitUsers(rootCmd)   // user creation, login, logout
	assets.InitAssets(rootCmd) // asset listing, create, update, delete
	scan.InitScan(rootCmd)     // network scan commands

	// Execute the CLI
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
