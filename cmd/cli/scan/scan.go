package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/crucial707/hci-asset/cmd/cli/config"
	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/spf13/cobra"
)

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
		runScanCmd(),
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

			req, _ := http.NewRequest("POST", config.APIURL()+"/scan", bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
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

			req, _ := http.NewRequest("GET", config.APIURL()+"/scan/"+jobID, nil)
			config.AddAuthHeader(req)

			resp, err := http.DefaultClient.Do(req)
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

			req, _ := http.NewRequest("POST", config.APIURL()+"/scan/"+jobID+"/cancel", nil)
			config.AddAuthHeader(req)

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

// ==========================
// Run Scan (start + poll)
// ==========================
func runScanCmd() *cobra.Command {
	var intervalSec int

	cmd := &cobra.Command{
		Use:   "run [target]",
		Short: "Start a scan and wait for it to complete",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			if intervalSec <= 0 {
				intervalSec = 3
			}

			// Start scan
			payload := map[string]string{"target": target}
			data, _ := json.Marshal(payload)

			startReq, _ := http.NewRequest("POST", config.APIURL()+"/scan", bytes.NewBuffer(data))
			startReq.Header.Set("Content-Type", "application/json")
			config.AddAuthHeader(startReq)

			startResp, err := http.DefaultClient.Do(startReq)
			if err != nil {
				fmt.Println("Failed to start scan:", err)
				return
			}
			defer startResp.Body.Close()

			if startResp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(startResp.Body)
				fmt.Printf("Failed to start scan: %s\n", string(body))
				return
			}

			var startResult struct {
				JobID  string `json:"job_id"`
				Status string `json:"status"`
			}
			if err := json.NewDecoder(startResp.Body).Decode(&startResult); err != nil {
				fmt.Println("Failed to parse start response:", err)
				return
			}

			fmt.Println("Scan started successfully.")
			fmt.Printf("Job ID: %s\n", startResult.JobID)

			// Poll status until complete/canceled/error
			for {
				statusReq, _ := http.NewRequest("GET", config.APIURL()+"/scan/"+startResult.JobID, nil)
				config.AddAuthHeader(statusReq)

				statusResp, err := http.DefaultClient.Do(statusReq)
				if err != nil {
					fmt.Println("Failed to get scan status:", err)
					return
				}

				if statusResp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(statusResp.Body)
					statusResp.Body.Close()
					fmt.Printf("Failed to get scan status: %s\n", string(body))
					return
				}

				var job scanJobResponse
				if err := json.NewDecoder(statusResp.Body).Decode(&job); err != nil {
					statusResp.Body.Close()
					fmt.Println("Failed to parse status response:", err)
					return
				}
				statusResp.Body.Close()

				fmt.Printf("Status: %s\n", job.Status)

				if job.Status == "running" {
					// Wait and poll again
					time.Sleep(time.Duration(intervalSec) * time.Second)
					continue
				}

				if job.Error != "" {
					fmt.Printf("Error: %s\n", job.Error)
				}

				if len(job.Assets) > 0 {
					fmt.Println("\nDiscovered Assets:")
					headers := []string{"ID", "Name", "Description", "Created At"}
					rows := make([][]interface{}, 0, len(job.Assets))
					for _, a := range job.Assets {
						rows = append(rows, []interface{}{
							a.ID,
							a.Name,
							a.Description,
							a.CreatedAt,
						})
					}
					output.RenderTable(headers, rows)
				} else {
					fmt.Println("\nNo assets discovered.")
				}

				break
			}
		},
	}

	cmd.Flags().IntVar(&intervalSec, "interval", 3, "Polling interval in seconds")
	return cmd
}
