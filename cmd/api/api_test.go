package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/config"
)

// TestAPI_LoginThenListAssets is an integration test: it builds the full router with a
// sqlmock-backed DB, logs in to get a JWT, then calls GET /assets with the token.
func TestAPI_LoginThenListAssets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	// Login: GetByUsername("integration")
	mock.ExpectQuery(`SELECT id, username`).
		WithArgs("integration").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username"}).AddRow(1, "integration"))

	// GET /assets: List(10, 0) default limit/offset
	mock.ExpectQuery(`SELECT id, name, description, last_seen FROM assets ORDER BY id LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "last_seen"}).
			AddRow(1, "asset1", "desc1", nil))

	cfg := config.Config{
		JWTSecret: "test-secret-for-integration",
		NmapPath:  "nmap",
	}
	r := newRouter(db, cfg)
	srv := httptest.NewServer(r)
	defer srv.Close()

	// 1) Login
	loginBody, _ := json.Marshal(map[string]string{"username": "integration"})
	loginResp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer loginResp.Body.Close()
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login status: got %d, want 200", loginResp.StatusCode)
	}
	var loginOut struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginOut); err != nil || loginOut.Token == "" {
		t.Fatalf("login response: %v", err)
	}

	// 2) GET /assets with Bearer token
	req, _ := http.NewRequest("GET", srv.URL+"/assets", nil)
	req.Header.Set("Authorization", "Bearer "+loginOut.Token)
	assetsResp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("assets request: %v", err)
	}
	defer assetsResp.Body.Close()
	if assetsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /assets status: got %d, want 200", assetsResp.StatusCode)
	}
	var assets []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(assetsResp.Body).Decode(&assets); err != nil {
		t.Fatalf("decode assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Name != "asset1" {
		t.Errorf("unexpected assets: %+v", assets)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

// TestAPI_Health is a quick smoke test for the health endpoint.
func TestAPI_Health(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	cfg := config.Config{JWTSecret: "x", NmapPath: "nmap"}
	r := newRouter(db, cfg)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status: got %d, want 200", resp.StatusCode)
	}
}

// TestAPI_Ready checks that /ready pings the DB and returns 200 when DB is reachable.
func TestAPI_Ready(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	cfg := config.Config{JWTSecret: "x", NmapPath: "nmap"}
	r := newRouter(db, cfg)
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/ready")
	if err != nil {
		t.Fatalf("ready request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /ready status: got %d, want 200", resp.StatusCode)
	}
}
