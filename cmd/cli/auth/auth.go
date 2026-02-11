package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/config"
	"github.com/spf13/cobra"
)

// InitAuth registers auth-related CLI commands (e.g., login) on the root command.
func InitAuth(rootCmd *cobra.Command) {
	rootCmd.AddCommand(loginCmd())
}

// loginCmd creates a command that logs in a user and stores the JWT token locally.
func loginCmd() *cobra.Command {
	var username string
	var register bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to the HCI Asset API",
		Long:  "Authenticate with the HCI Asset API and store a JWT token for subsequent CLI commands.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if username == "" {
				return fmt.Errorf("username is required")
			}

			client := http.DefaultClient

			// Optionally register the user first
			if register {
				if err := callJSONEndpoint(client, "/auth/register", map[string]string{"username": username}, nil); err != nil {
					return fmt.Errorf("failed to register user: %w", err)
				}
			}

			// Perform login to get token
			var loginResp struct {
				Token string `json:"token"`
			}
			if err := callJSONEndpoint(client, "/auth/login", map[string]string{"username": username}, &loginResp); err != nil {
				return fmt.Errorf("failed to login: %w", err)
			}
			if loginResp.Token == "" {
				return fmt.Errorf("login succeeded but no token returned")
			}

			if err := config.SaveToken(loginResp.Token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			fmt.Println("Login successful. Token stored locally.")
			return nil
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username to authenticate as")
	cmd.Flags().BoolVar(&register, "register", false, "Register the user before logging in")

	return cmd
}

func callJSONEndpoint(client *http.Client, path string, payload interface{}, out interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", config.APIURL()+path, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return err
		}
	}

	return nil
}

