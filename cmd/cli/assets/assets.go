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
		Short: "Manage assets",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all assets",
		RunE:  runList,
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a network for assets",
		RunE:  runScan,
	}
	scanCmd.Flags().StringP("target", "t", "", "Target network or IP (e.g., 192.168.1.0/24)")
	scanCmd.MarkFlagRequired("target")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get scan job status",
		RunE:  runStatus,
	}
	statusCmd.Flags().StringP("id", "i", "", "Scan job ID")
	statusCmd.MarkFlagRequired("id")

	assetsCmd.AddCommand(listCmd, scanCmd, statusCmd)

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

func runScan(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")
	target = strings.TrimSpace(target)

	resp, err := http.Post("http://localhost:8080/scan",
		"application/json",
		strings.NewReader(fmt.Sprintf(`{"target":"%s"}`, target)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("Scan started. Job ID: %v\n", result["job_id"])
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	jobID, _ := cmd.Flags().GetString("id")
	jobID = strings.TrimSpace(jobID)

	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/scan/%s", jobID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var job map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return err
	}

	fmt.Printf("Job Status: %v\n", job["status"])
	if assets, ok := job["assets"].([]interface{}); ok {
		fmt.Printf("Discovered %d assets\n", len(assets))
	}
	if errMsg, ok := job["error"].(string); ok && errMsg != "" {
		fmt.Printf("Error: %v\n", errMsg)
	}

	return nil
}
