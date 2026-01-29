package users

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var tokenFile = filepath.Join(os.TempDir(), "hci_token")

// ==========================
// Init Users
// ==========================
func InitUsers(rootCmd *cobra.Command) {

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "User operations",
	}

	rootCmd.AddCommand(usersCmd)
}

// ==========================
// Save Token
// ==========================
func SaveToken(token string) error {
	return os.WriteFile(tokenFile, []byte(token), 0600)
}

// ==========================
// Read Token
// ==========================
func ReadToken() (string, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", errors.New("not logged in")
	}
	return string(data), nil
}
