package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

var apiBase = "http://localhost:8080"

func init() {

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan network and ingest assets",
		RunE:  runScan,
	}

	scanCmd.Flags().String("target", "", "Network target (ex: 192.168.1.0/24)")
	scanCmd.MarkFlagRequired("target")

	root.RootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {

	target, _ := cmd.Flags().GetString("target")

	body, _ := json.Marshal(map[string]string{
		"target": target,
	})

	resp, err := http.Post(
		apiBase+"/scan",
		"application/json",
		bytes.NewBuffer(body),
	)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	fmt.Println("Scan started and assets ingested")

	return nil
}
