package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/crucial707/hci-asset/internal/models"
)

// captureOutput helps capture stdout during command execution.
func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestListUsers_TableOutput(t *testing.T) {
	users := []models.User{
		{ID: 1, Username: "alice"},
		{ID: 2, Username: "bob"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(users)
	}))
	defer srv.Close()

	_ = os.Setenv("HCI_ASSET_API_URL", srv.URL)
	defer os.Unsetenv("HCI_ASSET_API_URL")

	cmd := listUsersCmd()

	out := captureOutput(t, func() {
		cmd.Run(cmd, []string{})
	})

	if !strings.Contains(out, "alice") || !strings.Contains(out, "bob") {
		t.Fatalf("expected usernames in output, got: %s", out)
	}
}

func TestListUsers_JSONOutput(t *testing.T) {
	users := []models.User{
		{ID: 1, Username: "alice"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(users)
	}))
	defer srv.Close()

	_ = os.Setenv("HCI_ASSET_API_URL", srv.URL)
	defer os.Unsetenv("HCI_ASSET_API_URL")

	cmd := listUsersCmd()
	_ = cmd.Flags().Set("json", "true")

	out := captureOutput(t, func() {
		cmd.Run(cmd, []string{})
	})

	if !strings.Contains(out, `"username": "alice"`) {
		t.Fatalf("expected JSON output, got: %s", out)
	}
}

