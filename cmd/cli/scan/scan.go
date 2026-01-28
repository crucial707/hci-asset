package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

func init() {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Start a network scan to discover assets",
		Long: `Start a network scan using nmap on a target network or subnet.
The scan will run asynchronously and return a job ID that can be checked using "hci scan-status".

Example:
  hci scan --target 192.168.1.1/24`,
		RunE: runScan,
	}

	scanCmd.Flags().StringP("target", "t", "", "Target IP or subnet to scan (required)")
	scanCmd.Flags().BoolP("json", "j", false, "Output raw JSON instead of formatted text")
	scanCmd.MarkFlagRequired("target")

	root.GetRoot().AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	target, _ := cmd.Flags().GetString("target")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	payload := map[string]string{"target": target}
	bodyBytes, _ := json.Marshal(payload)

	resp, err := http.Post("http://localhost:8080/scan", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to call API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Printf("Scan started for target: %s\n", target)
		fmt.Printf("Job ID: %v\n", result["job_id"])
		fmt.Printf("Status: %v\n", result["status"])
		fmt.Println("Use 'hci scan-status --id <job_id>' to check progress.")
	}

	return nil
}
