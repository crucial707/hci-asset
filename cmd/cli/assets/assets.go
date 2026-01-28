package assets

import (
	"bytes"
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
  create    - Create a new asset
  update    - Update an existing asset
  delete    - Delete an asset by ID`,
	}

	// ===== LIST =====
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
	listCmd.Flags().Int("limit", 10, "Number of assets to return")
	listCmd.Flags().Int("offset", 0, "Pagination offset")
	listCmd.Flags().String("search", "", "Search filter")

	// ===== CREATE =====
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new asset",
		Long: `Create a new asset in the HCI system.

Required flags:
  --name         Name of the asset
  --description  Description of the asset`,
		RunE: runCreate,
	}
	createCmd.Flags().String("name", "", "Name of the asset")
	createCmd.Flags().String("description", "", "Description of the asset")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("description")

	// ===== UPDATE =====
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing asset",
		Long: `Update an existing asset in the HCI system.

Required flags:
  --id           ID of the asset to update
  --name         New name of the asset
  --description  New description of the asset`,
		RunE: runUpdate,
	}
	updateCmd.Flags().Int("id", 0, "ID of the asset")
	updateCmd.Flags().String("name", "", "New name of the asset")
	updateCmd.Flags().String("description", "", "New description of the asset")
	updateCmd.MarkFlagRequired("id")
	updateCmd.MarkFlagRequired("name")
	updateCmd.MarkFlagRequired("description")

	// ===== DELETE =====
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an asset by ID",
		Long: `Delete an asset from the system.

Required flags:
  --id   ID of the asset to delete`,
		RunE: runDelete,
	}
	deleteCmd.Flags().Int("id", 0, "ID of the asset")
	deleteCmd.MarkFlagRequired("id")

	// Add subcommands
	assetsCmd.AddCommand(listCmd, createCmd, updateCmd, deleteCmd)
	root.GetRoot().AddCommand(assetsCmd)
}

// ================= LIST =================
func runList(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt("limit")
	offset, _ := cmd.Flags().GetInt("offset")
	search, _ := cmd.Flags().GetString("search")

	url := fmt.Sprintf("http://localhost:8080/assets/list?limit=%d&offset=%d", limit, offset)
	if search != "" {
		url += "&search=" + search
	}

	resp, err := http.Get(url)
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

// ================= CREATE =================
func runCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	body := map[string]string{"name": name, "description": description}
	bodyJSON, _ := json.Marshal(body)

	resp, err := http.Post("http://localhost:8080/assets", "application/json", bytes.NewReader(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create asset: %v", err)
	}
	defer resp.Body.Close()

	var asset map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return fmt.Errorf("failed to parse create response: %v", err)
	}

	fmt.Printf("Asset created: ID=%v, Name=%v, Description=%v\n",
		asset["id"], asset["name"], asset["description"])
	return nil
}

// ================= UPDATE =================
func runUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt("id")
	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	body := map[string]string{"name": name, "description": description}
	bodyJSON, _ := json.Marshal(body)

	url := fmt.Sprintf("http://localhost:8080/assets/%d", id)
	req, _ := http.NewRequest("PUT", url, bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update asset: %v", err)
	}
	defer resp.Body.Close()

	var asset map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
		return fmt.Errorf("failed to parse update response: %v", err)
	}

	fmt.Printf("Asset updated: ID=%v, Name=%v, Description=%v\n",
		asset["id"], asset["name"], asset["description"])
	return nil
}

// ================= DELETE =================
func runDelete(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt("id")

	url := fmt.Sprintf("http://localhost:8080/assets/%d", id)
	req, _ := http.NewRequest("DELETE", url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete asset: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("failed to delete asset, status code: %d", resp.StatusCode)
	}

	fmt.Printf("Asset deleted: ID=%d\n", id)
	return nil
}
