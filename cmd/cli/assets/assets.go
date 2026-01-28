package assets

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/crucial707/hci-asset/cmd/cli/users"
	"github.com/spf13/cobra"
)

func init() {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage assets in HCI Asset Management API",
		Long: `Assets allow you to create, list, update, or delete devices
discovered in your network or manually added.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		Long:  "Fetches all assets from the API, supports pagination and search (via API).",
		RunE:  runList,
	}

	assetsCmd.AddCommand(listCmd)
	root.GetRoot().AddCommand(assetsCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Load JWT token from user store
	token, err := users.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token: %v. Have you logged in?", err)
	}

	req, err := http.NewRequest("GET", "http://localhost:8080/assets/list", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
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
