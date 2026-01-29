package scan

import "github.com/spf13/cobra"

// ==========================
// Init Scan (stub)
// ==========================
func InitScan(rootCmd *cobra.Command) {

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Network scanning (coming soon)",
	}

	rootCmd.AddCommand(scanCmd)
}
