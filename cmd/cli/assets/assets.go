package assets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

func init() {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets in the HCI system",
		Long: `The assets command allows you to interact with assets stored in the HCI system database.

Available subcommands:
  list      - List all assets
  create    - Create a new asset (requires API JWT auth)
  update    - Update an existing asset (requires API JWT auth)
  delete    - Delete an asset by ID (requires API JWT auth)`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		Long: `List all assets in the system.

Optional flags:
  --limit    Number of assets per page (default: 10)
  --offset   Pagination offset (default: 0)
  --search   Filter assets by name or description containing this string`,
		RunE: runList,
	}

	assetsCmd.AddCommand(listCmd)
	root.GetRoot().AddCommand(assetsCmd)
}

// runList calls the API and lists assets
func runList(cmd *cobra.Command, args []string) error {
	resp, err := http.Get("http://localhost:8080/assets/list")
	if err != nil {
		return fmt.Errorf("failed to fetch assets: %v", err)
	}
	defer resp.Body.Close()

	var assets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return fmt.Errorf("failed to decode assets response: %v", err)
	}

	if len(assets) == 0 {
		fmt.Println("No assets found.")
		return nil
	}

	fmt.Printf("%-5s %-30s %-50s %-25s\n", "ID", "Name", "Description", "CreatedAt")
	fmt.Println(strings.Repeat("-", 120))
	for _, a := range assets {
		fmt.Printf("%-5v %-30v %-50v %-25v\n",
			a["id"], a["name"], a["description"], a["created_at"])
	}
	return nil
}
