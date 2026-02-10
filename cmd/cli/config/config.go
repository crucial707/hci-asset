package config

import "os"

const defaultAPIURL = "http://localhost:8080"

// APIURL returns the base URL for the HCI Asset API.
// It can be overridden with the HCI_ASSET_API_URL environment variable.
func APIURL() string {
	if v := os.Getenv("HCI_ASSET_API_URL"); v != "" {
		return v
	}
	return defaultAPIURL
}

