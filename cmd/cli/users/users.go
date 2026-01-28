package users

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/crucial707/hci-asset/cmd/cli/root"
	"github.com/spf13/cobra"
)

const tokenFileName = ".hci_token"

// ==========================
// CLI Command Init
// ==========================
func init() {
	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users and authentication",
		Long: `Register or login a user to the HCI Asset Management API.
Stores JWT token locally for future commands.`,
	}

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new user",
		Long:  "Register a new user with username and password.",
		RunE:  runRegister,
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login an existing user",
		Long:  "Login and save JWT token locally for future CLI commands.",
		RunE:  runLogin,
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout current user",
		Long:  "Remove locally saved JWT token.",
		RunE:  runLogout,
	}

	usersCmd.AddCommand(registerCmd, loginCmd, logoutCmd)
	root.GetRoot().AddCommand(usersCmd)
}

// ==========================
// Register User
// ==========================
func runRegister(cmd *cobra.Command, args []string) error {
	var username, password string
	fmt.Print("Username: ")
	fmt.Scanln(&username)
	fmt.Print("Password: ")
	fmt.Scanln(&password)

	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(b))
	}

	fmt.Println("User registered successfully! You can now login.")
	return nil
}

// ==========================
// Login User
// ==========================
func runLogin(cmd *cobra.Command, args []string) error {
	var username, password string
	fmt.Print("Username: ")
	fmt.Scanln(&username)
	fmt.Print("Password: ")
	fmt.Scanln(&password)

	payload := map[string]string{
		"username": username,
		"password": password,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(b))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	token, ok := result["token"]
	if !ok {
		return fmt.Errorf("token not returned by API")
	}

	if err := saveToken(token); err != nil {
		return err
	}

	fmt.Println("Login successful! JWT token saved locally.")
	return nil
}

// ==========================
// Logout User
// ==========================
func runLogout(cmd *cobra.Command, args []string) error {
	path := tokenPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Println("No user logged in.")
		return nil
	}

	if err := os.Remove(path); err != nil {
		return err
	}

	fmt.Println("Logged out successfully.")
	return nil
}

// ==========================
// Token Storage Helpers
// ==========================
func saveToken(token string) error {
	path := tokenPath()
	return ioutil.WriteFile(path, []byte(token), 0600)
}

func LoadToken() (string, error) {
	path := tokenPath()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func tokenPath() string {
	dir, _ := os.UserHomeDir()
	return filepath.Join(dir, tokenFileName)
}
