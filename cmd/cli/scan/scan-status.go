package scan

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd := &cobra.Command{
		Use:   "scan-status",
		Short: "Check the status of a network scan",
		Long: `Check the status of an ongoing or completed network scan.
Provide the job ID returned when starting a scan with "hci scan".

Example:
  hci scan-status --id 1`,
		RunE: runScanStatus,
	}

	statusCmd.Flags().StringP("id", "i", "", "Scan job ID (required)")
	statusCmd.Flags().BoolP("json", "j", false, "Output raw JSON instead of formatted text")
	statusCmd.MarkFlagRequired("id")

	root.GetRoot().AddCommand(statusCmd)
}

func runScanStatus(cmd *cobra.Command, args []string) error {
	jobID, _ := cmd.Flags().GetString("id")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	url := fmt.Sprintf("http://localhost:8080/scan/%s", jobID)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to call API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(bodyBytes))
	}

	var result struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if jsonOutput {
		out, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(out))
		return nil
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

	return nil
}
