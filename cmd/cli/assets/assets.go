package assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"

	"github.com/crucial707/hci-asset/cmd/cli/config"
	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/crucial707/hci-asset/internal/handlers"
	"github.com/crucial707/hci-asset/internal/models"
)

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
		heartbeatAssetCmd(),
		deleteAssetCmd(),
	)

	rootCmd.AddCommand(assetsCmd)
}

// ==========================
// List Assets (Pretty Table)
// ==========================
func listAssetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		Run: func(cmd *cobra.Command, args []string) {
			req, _ := http.NewRequest("GET", config.APIURL()+"/assets", nil)
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
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

			// Optional JSON output for scripting
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				out, err := json.MarshalIndent(assets, "", "  ")
				if err != nil {
					fmt.Println("Failed to encode JSON:", err)
					return
				}
				fmt.Println(string(out))
				return
			}

			headers := []string{"ID", "Name", "Description", "Last seen"}

			rows := [][]interface{}{}
			for _, a := range assets {
				lastSeen := "Never"
				if a.LastSeen != nil {
					lastSeen = a.LastSeen.Format(time.RFC3339)
				}
				rows = append(rows, []interface{}{
					a.ID,
					a.Name,
					a.Description,
					lastSeen,
				})
			}

			output.RenderTable(headers, rows)
		},
	}

	cmd.Flags().BoolP("json", "j", false, "Output raw JSON instead of formatted text")
	return cmd
}

// ==========================
// Heartbeat Asset (update last_seen)
// ==========================
func heartbeatAssetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat [id]",
		Short: "Record a heartbeat for an asset (updates last_seen)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			req, _ := http.NewRequest("POST", config.APIURL()+"/assets/"+id+"/heartbeat", nil)
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to record heartbeat (%d): %s\n", resp.StatusCode, string(body))
				return
			}

			var asset models.Asset
			if err := json.NewDecoder(resp.Body).Decode(&asset); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}
			lastSeen := "Never"
			if asset.LastSeen != nil {
				lastSeen = asset.LastSeen.Format(time.RFC3339)
			}
			fmt.Printf("Heartbeat recorded for asset %s (id %d). Last seen: %s\n", asset.Name, asset.ID, lastSeen)
		},
	}
	return cmd
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
			req, _ := http.NewRequest("POST", config.APIURL()+"/assets", bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to create asset (%d): %s\n", resp.StatusCode, string(body))
				return
			}

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
			req, _ := http.NewRequest("PUT", config.APIURL()+"/assets/"+id, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to update asset (%d): %s\n", resp.StatusCode, string(body))
				return
			}

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

			req, _ := http.NewRequest("DELETE", config.APIURL()+"/assets/"+id, nil)
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNoContent {
				fmt.Println("Asset deleted successfully")
				return
			}

			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("Failed to delete asset (%d): %s\n", resp.StatusCode, string(body))
		},
	}
}
