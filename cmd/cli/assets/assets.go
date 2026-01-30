package assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/models"
)

// ==========================
// CLI API URL
// ==========================
var apiURL = "http://localhost:8080"

// ==========================
// Initialize Assets CLI
// ==========================
func InitAssets(rootCmd *cobra.Command, assetHandler *handlers.AssetHandler) {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets",
		Long:  "Commands to list, create, update, and delete assets via the API.",
	}

	assetsCmd.AddCommand(
		listAssetsCmd(),
		createAssetCmd(),
		updateAssetCmd(),
		deleteAssetCmd(),
	)

	rootCmd.AddCommand(assetsCmd)
}

// ==========================
// List Assets (Pretty Table)
// ==========================
func listAssetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(apiURL + "/assets")
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			var assets []models.Asset
			if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}

			if len(assets) == 0 {
				fmt.Println("No assets found.")
				return
			}

			headers := []string{"ID", "Name", "Description"}

			rows := [][]interface{}{}
			for _, a := range assets {
				rows = append(rows, []interface{}{
					a.ID,
					a.Name,
					a.Description,
				})
			}

			output.RenderTable(headers, rows)
		},
	}
}

// ==========================
// Create Asset
// ==========================
func createAssetCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new asset",
		Run: func(cmd *cobra.Command, args []string) {
			if name == "" || description == "" {
				fmt.Println("name and description are required")
				return
			}

			payload := map[string]string{
				"name":        name,
				"description": description,
			}
			data, _ := json.Marshal(payload)

			resp, err := http.Post(apiURL+"/assets", "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Asset name")
	cmd.Flags().StringVar(&description, "description", "", "Asset description")
	return cmd
}

// ==========================
// Update Asset
// ==========================
func updateAssetCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update an asset",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if name == "" && description == "" {
				fmt.Println("Provide --name or --description to update")
				return
			}

			payload := map[string]string{}
			if name != "" {
				payload["name"] = name
			}
			if description != "" {
				payload["description"] = description
			}

			data, _ := json.Marshal(payload)
			req, _ := http.NewRequest("PUT", apiURL+"/assets/"+id, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New asset name")
	cmd.Flags().StringVar(&description, "description", "", "New asset description")
	return cmd
}

// ==========================
// Delete Asset
// ==========================
func deleteAssetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete an asset",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]

			req, _ := http.NewRequest("DELETE", apiURL+"/assets/"+id, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 204 {
				fmt.Println("Asset deleted successfully")
			} else {
				body, _ := io.ReadAll(resp.Body)
				fmt.Println("Failed to delete asset:", string(body))
			}
		},
	}
}
