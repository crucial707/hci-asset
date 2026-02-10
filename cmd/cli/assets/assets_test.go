package assets

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

func TestListAssets_TableOutput(t *testing.T) {
	assets := []models.Asset{
		{ID: 1, Name: "asset-1", Description: "first"},
		{ID: 2, Name: "asset-2", Description: "second"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(assets)
	}))
	defer srv.Close()

	_ = os.Setenv("HCI_ASSET_API_URL", srv.URL)
	defer os.Unsetenv("HCI_ASSET_API_URL")

	cmd := listAssetsCmd()

	out := captureOutput(t, func() {
		cmd.Run(cmd, []string{})
	})

	if !strings.Contains(out, "asset-1") || !strings.Contains(out, "asset-2") {
		t.Fatalf("expected asset names in output, got: %s", out)
	}
}

func TestListAssets_JSONOutput(t *testing.T) {
	assets := []models.Asset{
		{ID: 1, Name: "asset-1", Description: "first"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(assets)
	}))
	defer srv.Close()

	_ = os.Setenv("HCI_ASSET_API_URL", srv.URL)
	defer os.Unsetenv("HCI_ASSET_API_URL")

	cmd := listAssetsCmd()
	_ = cmd.Flags().Set("json", "true")

	out := captureOutput(t, func() {
		cmd.Run(cmd, []string{})
	})

	if !strings.Contains(out, `"name": "asset-1"`) {
		t.Fatalf("expected JSON output, got: %s", out)
	}
}

