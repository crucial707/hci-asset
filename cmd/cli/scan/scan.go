package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// ==========================
// CLI API URL
// ==========================
var apiURL = "http://localhost:8080"

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
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
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
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
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
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}
}
