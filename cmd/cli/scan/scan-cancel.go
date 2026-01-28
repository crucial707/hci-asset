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
	cancelCmd := &cobra.Command{
		Use:   "scan-cancel",
		Short: "Cancel an ongoing network scan",
		Long: `Cancel a running network scan by providing the job ID.
Use this command if you want to stop a scan before it completes.

Example:
  hci scan-cancel --id 1`,
		RunE: runScanCancel,
	}

	cancelCmd.Flags().StringP("id", "i", "", "Scan job ID to cancel (required)")
	cancelCmd.Flags().BoolP("json", "j", false, "Output raw JSON instead of formatted text")
	cancelCmd.MarkFlagRequired("id")

	root.GetRoot().AddCommand(cancelCmd)
}

func runScanCancel(cmd *cobra.Command, args []string) error {
	jobID, _ := cmd.Flags().GetString("id")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	url := fmt.Sprintf("http://localhost:8080/scan/%s", jobID)
	req, err := http.NewRequest("DELETE", url, bytes.NewReader(nil))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call API: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("API error: %s", string(bodyBytes))
	}

	if jsonOutput {
		var result map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Printf("Scan job %s has been canceled successfully.\n", jobID)
	}

	return nil
}
