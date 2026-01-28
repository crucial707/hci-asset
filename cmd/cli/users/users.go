package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var authToken string // stores JWT after login

func InitUsers(rootCmd *cobra.Command) {
	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users (create and login)",
		Long: `User management commands:
- create: create a new user with username and password
- login: authenticate and receive a JWT token`,
	}

	// -----------------------
	// Create User Command
	// -----------------------
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new user",
		RunE:  runCreateUser,
	}
	createCmd.Flags().StringP("username", "u", "", "Username for new user (required)")
	createCmd.Flags().StringP("password", "p", "", "Password for new user (required)")
	createCmd.MarkFlagRequired("username")
	createCmd.MarkFlagRequired("password")

	// -----------------------
	// Login Command
	// -----------------------
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login as a user and obtain JWT token",
		RunE:  runLoginUser,
	}
	loginCmd.Flags().StringP("username", "u", "", "Username (required)")
	loginCmd.Flags().StringP("password", "p", "", "Password (required)")
	loginCmd.MarkFlagRequired("username")
	loginCmd.MarkFlagRequired("password")

	usersCmd.AddCommand(createCmd, loginCmd)
	rootCmd.AddCommand(usersCmd)
}

// -----------------------
// Run Create User
// -----------------------
func runCreateUser(cmd *cobra.Command, args []string) error {
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	input := UserInput{Username: username, Password: password}
	bodyBytes, _ := json.Marshal(input)

	resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Response:", string(respBody))
	return nil
}

// -----------------------
// Run Login User
// -----------------------
func runLoginUser(cmd *cobra.Command, args []string) error {
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	input := UserInput{Username: username, Password: password}
	bodyBytes, _ := json.Marshal(input)

	resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var respJSON map[string]string
	json.NewDecoder(resp.Body).Decode(&respJSON)

	token, ok := respJSON["token"]
	if !ok {
		fmt.Println("Login failed:", string(bodyBytes))
		return nil
	}

	authToken = token
	fmt.Println("Login successful! JWT token stored for this session.")
	return nil
}

// -----------------------
// Helper to attach JWT to requests
// -----------------------
func AuthHeader(req *http.Request) {
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
}
