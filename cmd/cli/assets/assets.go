package assets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

func init() {
	// Parent command
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets",
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		RunE:  runList,
	}

	assetsCmd.AddCommand(listCmd)

	// Attach to root command
	root.RootCmd.AddCommand(assetsCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	resp, err := http.Get("http://localhost:8080/assets/list")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var assets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return err
	}

	fmt.Println("Assets:")
	for _, a := range assets {
		fmt.Printf("- ID: %v, Name: %v, Description: %v, CreatedAt: %v\n",
			a["id"], a["name"], a["description"], a["created_at"])
	}

	return nil
}
