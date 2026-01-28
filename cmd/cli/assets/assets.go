package assets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/users"
	"github.com/spf13/cobra"
)

func InitAssets(rootCmd *cobra.Command) {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets",
		Long:  "List assets or fetch asset details from the HCI Asset Management API",
	}

	// -----------------------
	// List Assets Command
	// -----------------------
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		Long: `List all assets currently in the database.
Requires user login to attach JWT token.`,
		RunE: runList,
	}

	assetsCmd.AddCommand(listCmd)
	rootCmd.AddCommand(assetsCmd)
}

// -----------------------
// Run List Assets
// -----------------------
func runList(cmd *cobra.Command, args []string) error {
	req, err := http.NewRequest("GET", "http://localhost:8080/assets/list", nil)
	if err != nil {
		return err
	}

	// Attach JWT if logged in
	users.AuthHeader(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Error:", string(body))
		return nil
	}

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
