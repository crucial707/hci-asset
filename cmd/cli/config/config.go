package config

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultAPIURL  = "http://localhost:8080/v1"
	tokenDirName   = ".hci-asset"
	tokenFileName  = "token"
	tokenEnvVar    = "HCI_ASSET_TOKEN"
	apiURLEnvVar   = "HCI_ASSET_API_URL"
)

// APIURL returns the base URL for the HCI Asset API.
// It can be overridden with the HCI_ASSET_API_URL environment variable.
func APIURL() string {
	if v := os.Getenv(apiURLEnvVar); v != "" {
		return v
	}
	return defaultAPIURL
}

// Token returns the current auth token, if any, from env or the local token file.
func Token() string {
	if v := os.Getenv(tokenEnvVar); v != "" {
		return v
	}

	path, err := tokenFilePath()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

// SaveToken persists the auth token to a local file in the user's home directory.
func SaveToken(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(strings.TrimSpace(token)), 0o600)
}

// AddAuthHeader adds the Authorization header to the request if a token is available.
func AddAuthHeader(req *http.Request) {
	if req == nil {
		return
	}
	if token := Token(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, tokenDirName, tokenFileName), nil
}
