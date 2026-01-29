package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

// ==========================
// CLI API URL
// ==========================
var apiURL = "http://localhost:8080"

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
// List Users
// ==========================
func listUsersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all users",
		Run: func(cmd *cobra.Command, args []string) {
			resp, err := http.Get(apiURL + "/users")
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			fmt.Println(string(body))
		},
	}
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

			resp, err := http.Post(apiURL+"/users", "application/json", bytes.NewBuffer(data))
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()
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
			req, _ := http.NewRequest("PUT", apiURL+"/users/"+id, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()
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

			req, _ := http.NewRequest("DELETE", apiURL+"/users/"+id, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println("API request failed:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == 204 {
				fmt.Println("User deleted successfully")
			} else {
				body, _ := io.ReadAll(resp.Body)
				fmt.Println("Failed to delete user:", string(body))
			}
		},
	}
}
