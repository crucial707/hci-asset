package main

import (
	"fmt"
	"os"

	"github.com/crucial707/hci-asset/cmd/cli/root"
)

func main() {
	fmt.Println("Command line interface for interacting with HCI Asset Management API")

	// Execute the root Cobra command
	if err := root.GetRoot().Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
