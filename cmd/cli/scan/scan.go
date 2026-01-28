package scan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

func init() {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a network for devices",
		Long: `Scan a network or subnet to discover devices automatically.

Required flags:
  --target, -t    Target subnet or IP (e.g., 192.168.1.1/24)

The scan will trigger an Nmap job on the server and display discovered assets when complete.`,
		RunE: runScan,
	}

	scanCmd.Flags().StringP("target", "t", "", "Target subnet or IP (e.g., 192.168.1.1/24)")
	scanCmd.MarkFlagRequired("target")

	root.GetRoot().AddCommand(scanCmd)
}

// runScan triggers the scan API and polls until complete
func runScan(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("target is required")
	}

	body := map[string]string{"target": target}
	bodyJSON, _ := json.Marshal(body)

	resp, err := http.Post("http://localhost:8080/scan", "application/json", strings.NewReader(string(bodyJSON)))
	if err != nil {
		return fmt.Errorf("failed to start scan: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse scan response: %v", err)
	}

	jobID, ok := result["job_id"].(string)
	if !ok {
		return fmt.Errorf("invalid response from server")
	}

	fmt.Printf("Scan started. Job ID: %v\n", jobID)
	fmt.Println("Waiting for scan to complete...")

	var statusResp map[string]interface{}
	for {
		time.Sleep(2 * time.Second)
		statusResp, err = fetchScanStatus(jobID)
		if err != nil {
			return err
		}

		status, _ := statusResp["status"].(string)
		if status == "complete" || status == "error" {
			break
		}
		fmt.Print(".")
	}
	fmt.Println("\nScan completed.")

	if statusResp["status"] == "error" {
		fmt.Println("Scan error:", statusResp["error"])
		return nil
	}

	// Display discovered assets
	assetsRaw, ok := statusResp["assets"].([]interface{})
	if !ok || len(assetsRaw) == 0 {
		fmt.Println("No assets discovered.")
		return nil
	}

	fmt.Printf("%-5s %-30s %-50s\n", "ID", "Name", "Description")
	fmt.Println(strings.Repeat("-", 90))
	for _, a := range assetsRaw {
		if m, ok := a.(map[string]interface{}); ok {
			fmt.Printf("%-5v %-30v %-50v\n",
				m["id"], m["name"], m["description"])
		}
	}

	return nil
}

// fetchScanStatus polls the API for job status
func fetchScanStatus(jobID string) (map[string]interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/scan/%s", jobID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scan status: %v", err)
	}
	defer resp.Body.Close()

	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %v", err)
	}
	return status, nil
}
