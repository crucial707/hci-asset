package assets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/users"
	"github.com/spf13/cobra"
)

var apiURL = "http://localhost:8080"

// ==========================
// Init Assets
// ==========================
func InitAssets(rootCmd *cobra.Command) {

	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets",
	}

	assetsCmd.AddCommand(
		listAssetsCmd(),
		createAssetCmd(),
		deleteAssetCmd(),
	)

	rootCmd.AddCommand(assetsCmd)
}

// ==========================
// LIST
// ==========================
func listAssetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List assets",
		Run: func(cmd *cobra.Command, args []string) {

			token, err := users.ReadToken()
			if err != nil {
				fmt.Println("Please login first")
				return
			}

			req, _ := http.NewRequest("GET", apiURL+"/assets", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()

			var out any
			json.NewDecoder(resp.Body).Decode(&out)
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
		},
	}
}

// ==========================
// CREATE
// ==========================
func createAssetCmd() *cobra.Command {

	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create asset",
		Run: func(cmd *cobra.Command, args []string) {

			token, err := users.ReadToken()
			if err != nil {
				fmt.Println("Please login first")
				return
			}

			payload := map[string]string{
				"name":        name,
				"description": description,
			}

			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest("POST", apiURL+"/assets", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer resp.Body.Close()

			var out any
			json.NewDecoder(resp.Body).Decode(&out)
			b, _ := json.MarshalIndent(out, "", "  ")
			fmt.Println(string(b))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "asset name")
	cmd.Flags().StringVar(&description, "description", "", "asset description")

	return cmd
}

// ==========================
// DELETE
// ==========================
func deleteAssetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete asset",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			token, err := users.ReadToken()
			if err != nil {
				fmt.Println("Please login first")
				return
			}

			req, _ := http.NewRequest("DELETE", apiURL+"/assets/"+args[0], nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err)
				return
			}

			if resp.StatusCode == 204 {
				fmt.Println("Asset deleted")
			} else {
				fmt.Println("Failed to delete asset")
			}
		},
	}
}
