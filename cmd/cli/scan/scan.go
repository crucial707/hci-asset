package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/crucial707/hci-asset/cmd/cli/users"
	"github.com/spf13/cobra"
)

func init() {
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a network for devices",
		Long: `Perform a network scan using the API's Nmap integration.
Example usage:
  hci scan 192.168.1.0/24
This will start an async scan and return a job ID.`,
	}

	startCmd := &cobra.Command{
		Use:   "start [target]",
		Short: "Start a network scan",
		Long:  "Starts a scan for a given target network (CIDR or IP). Requires login.",
		Args:  cobra.ExactArgs(1),
		RunE:  runScan,
	}

	statusCmd := &cobra.Command{
		Use:   "status [job_id]",
		Short: "Check the status of a scan job",
		Long:  "Check the progress or result of an async scan using the job ID returned when the scan was started.",
		Args:  cobra.ExactArgs(1),
		RunE:  runStatus,
	}

	scanCmd.AddCommand(startCmd, statusCmd)
	root.GetRoot().AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	target := args[0]

	token, err := users.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token: %v. Have you logged in?", err)
	}

	payload := map[string]string{"target": target}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "http://localhost:8080/scan", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	fmt.Printf("Scan started: job_id=%v\n", result["job_id"])
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	jobID := args[0]

	token, err := users.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token: %v. Have you logged in?", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8080/scan/%s", jobID), nil)
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

	var job map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return err
	}

	fmt.Printf("Scan Job Status: %v\n", job)
	return nil
}
