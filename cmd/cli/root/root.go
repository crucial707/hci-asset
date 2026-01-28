package root

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Exported RootCmd
var RootCmd = &cobra.Command{
	Use:   "hci",
	Short: "HCI Asset Management CLI",
	Long:  "Command line interface for interacting with HCI Asset Management API",
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Optional helper to return the RootCmd
func GetRoot() *cobra.Command {
	return RootCmd
}
