package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/crucial707/hci-asset/cmd/cli/config"
	"github.com/crucial707/hci-asset/cmd/cli/output"
	"github.com/crucial707/hci-asset/internal/models"
	"github.com/spf13/cobra"
)

// ==========================
// Initialize Users CLI
// ==========================
func InitUsers(rootCmd *cobra.Command) {
	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
		Long:  "Commands to create, list, update, and delete users via the API.",
	}

	usersCmd.AddCommand(
		createUserCmd(),
		listUsersCmd(),
		updateUserCmd(),
		deleteUserCmd(),
	)

	rootCmd.AddCommand(usersCmd)
}

// ==========================
// List Users (Pretty Table)
// ==========================
func listUsersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(config.APIURL() + "/users")
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to list users: %s\n", string(body))
				return
			}

			var users []models.User
			if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
				fmt.Println("Failed to parse response:", err)
				return
			}

			if len(users) == 0 {
				fmt.Println("No users found.")
				return
			}

			// Optional JSON output for scripting
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				out, err := json.MarshalIndent(users, "", "  ")
				if err != nil {
					fmt.Println("Failed to encode JSON:", err)
					return
				}
				fmt.Println(string(out))
				return
			}

			headers := []string{"ID", "Username"}
			rows := make([][]interface{}, 0, len(users))
			for _, u := range users {
				rows = append(rows, []interface{}{
					u.ID,
					u.Username,
				})
			}

			output.RenderTable(headers, rows)
		},
	}

	cmd.Flags().BoolP("json", "j", false, "Output raw JSON instead of formatted text")
	return cmd
}

// ==========================
// Create User
// ==========================
func createUserCmd() *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		Run: func(cmd *cobra.Command, args []string) {
			if username == "" || password == "" {
				fmt.Println("username and password are required")
				return
			}

			payload := map[string]string{
				"username": username,
				"password": password,
			}
			data, _ := json.Marshal(payload)

			resp, err := http.Post(config.APIURL()+"/users", "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to create user (%d): %s\n", resp.StatusCode, string(body))
				return
			}

			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "Username")
	cmd.Flags().StringVar(&password, "password", "", "Password")
	return cmd
}

// ==========================
// Update User
// ==========================
func updateUserCmd() *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:   "update [id]",
		Short: "Update a user",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if username == "" && password == "" {
				fmt.Println("Provide --username or --password to update")
				return
			}

			payload := map[string]string{}
			if username != "" {
				payload["username"] = username
			}
			if password != "" {
				payload["password"] = password
			}

			data, _ := json.Marshal(payload)
			req, _ := http.NewRequest("PUT", config.APIURL()+"/users/"+id, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("Failed to update user (%d): %s\n", resp.StatusCode, string(body))
				return
			}

			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "New username")
	cmd.Flags().StringVar(&password, "password", "", "New password")
	return cmd
}

// ==========================
// Delete User
// ==========================
func deleteUserCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a user",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]

			req, _ := http.NewRequest("DELETE", config.APIURL()+"/users/"+id, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNoContent {
				fmt.Println("User deleted successfully")
			} else {
				body, _ := io.ReadAll(resp.Body)
				fmt.Println("Failed to delete user:", string(body))
			}
		},
	}
}
