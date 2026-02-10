package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/spf13/cobra"
)

// ==========================
// CLI API URL
// ==========================
var apiURL = "http://localhost:8080"

// ==========================
// Types
// ==========================
type scanJobResponse struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Assets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
	} `json:"assets"`
	Error string `json:"error,omitempty"`
}

// ==========================
// Initialize Scan CLI
// ==========================
func InitScan(rootCmd *cobra.Command) {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Manage network scans",
	}

	scanCmd.AddCommand(
		startScanCmd(),
		statusScanCmd(),
		cancelScanCmd(),
	)

	rootCmd.AddCommand(scanCmd)
}

// ==========================
// Start Scan
// ==========================
func startScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start [target]",
		Short: "Start a scan on a target",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			payload := map[string]string{"target": target}
			data, _ := json.Marshal(payload)

			resp, err := http.Post(apiURL+"/scan", "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Println("Failed to start scan:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to start scan: %s\n", string(body))
				return
			}

			var result struct {
				JobID  string `json:"job_id"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}

			fmt.Println("Scan started successfully.")
			fmt.Printf("Job ID: %s\n", result.JobID)
			fmt.Printf("Status: %s\n", result.Status)
		},
	}
}

// ==========================
// Get Scan Status
// ==========================
func statusScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [jobID]",
		Short: "Check scan status",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jobID := args[0]
			resp, err := http.Get(apiURL + "/scan/" + jobID)
			if err != nil {
				fmt.Println("Failed to get scan status:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to get scan status: %s\n", string(body))
				return
			}

			var result scanJobResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}

			fmt.Printf("Scan Job ID: %s\n", jobID)
			fmt.Printf("Target: %s\n", result.Target)
			fmt.Printf("Status: %s\n", result.Status)
			if result.Error != "" {
				fmt.Printf("Error: %s\n", result.Error)
			}

			if len(result.Assets) > 0 {
				fmt.Println("\nDiscovered Assets:")
				headers := []string{"ID", "Name", "Description", "Created At"}
				rows := make([][]interface{}, 0, len(result.Assets))
				for _, a := range result.Assets {
					rows = append(rows, []interface{}{
						a.ID,
						a.Name,
						a.Description,
						a.CreatedAt,
					})
				}
				output.RenderTable(headers, rows)
			} else {
				fmt.Println("\nNo assets discovered yet.")
			}
		},
	}
}

// ==========================
// Cancel Scan
// ==========================
func cancelScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel [jobID]",
		Short: "Cancel a running scan",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			jobID := args[0]

			req, _ := http.NewRequest("POST", apiURL+"/scan/"+jobID+"/cancel", nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("Failed to cancel scan:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to cancel scan: %s\n", string(body))
				return
			}

			var result scanJobResponse
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}

			fmt.Printf("Scan Job ID: %s\n", jobID)
			fmt.Printf("Target: %s\n", result.Target)
			fmt.Printf("Status: %s\n", result.Status)
			if result.Error != "" {
				fmt.Printf("Error: %s\n", result.Error)
			}

			if len(result.Assets) > 0 {
				fmt.Println("\nAssets discovered before cancellation:")
				headers := []string{"ID", "Name", "Description", "Created At"}
				rows := make([][]interface{}, 0, len(result.Assets))
				for _, a := range result.Assets {
					rows = append(rows, []interface{}{
						a.ID,
						a.Name,
						a.Description,
						a.CreatedAt,
					})
				}
				output.RenderTable(headers, rows)
			} else {
				fmt.Println("\nNo assets discovered for this job.")
			}
		},
	}
}
