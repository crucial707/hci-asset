package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/users"
	"github.com/spf13/cobra"
)

type ScanRequest struct {
	Target string `json:"target"`
}

// InitScan initializes the scan command in the CLI
func InitScan(rootCmd *cobra.Command) {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Perform network scans",
		Long: `Scan the network for hosts using Nmap.

Requires user login. Use JWT authentication from the login session.`,
	}

	// -----------------------
	// Run Scan
	// -----------------------
	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run a scan on a target (IP or subnet)",
		Long:  "Run a network scan against a specified IP or subnet (e.g., 192.168.1.0/24).",
		RunE:  runScan,
	}
	runCmd.Flags().StringP("target", "t", "", "Target IP address or subnet (required)")
	runCmd.MarkFlagRequired("target")

	// -----------------------
	// Get Scan Status
	// -----------------------
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check the status of a scan job",
		Long:  "Get the current status of a scan job by Job ID.",
		RunE:  getScanStatus,
	}
	statusCmd.Flags().StringP("id", "i", "", "Job ID to check (required)")
	statusCmd.MarkFlagRequired("id")

	scanCmd.AddCommand(runCmd, statusCmd)
	rootCmd.AddCommand(scanCmd)
}

// -----------------------
// Run Scan Command
// -----------------------
func runScan(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")
	reqBody := ScanRequest{Target: target}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "http://localhost:8080/scan", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	users.AuthHeader(req) // attach JWT

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

	var respJSON map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&respJSON)

	fmt.Printf("Scan job started! Job ID: %v, Status: %v\n", respJSON["job_id"], respJSON["status"])
	return nil
}

// -----------------------
// Get Scan Status Command
// -----------------------
func getScanStatus(cmd *cobra.Command, args []string) error {
	jobID, _ := cmd.Flags().GetString("id")

	url := "http://localhost:8080/scan/" + jobID
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	users.AuthHeader(req) // attach JWT

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

	var job map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&job)

	fmt.Printf("Job ID: %v\nStatus: %v\nDiscovered Assets: %v\n",
		job["target"], job["status"], len(job["assets"].([]interface{})))

	// Optionally print discovered assets
	assets := job["assets"].([]interface{})
	for i, a := range assets {
		assetMap := a.(map[string]interface{})
		fmt.Printf("%d. Name: %v, Description: %v\n", i+1, assetMap["name"], assetMap["description"])
	}

	return nil
}
