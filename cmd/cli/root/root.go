package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Exported RootCmd so other packages can attach subcommands
var RootCmd = &cobra.Command{
	Use:   "hci",
	Short: "HCI Asset Management CLI",
	Long:  "Command line interface for interacting with HCI Asset Management API",
}

// Execute runs the CLI
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func GetRoot() *cobra.Command {
	return RootCmd
}
