package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
)

// requestWithChiURLParams returns a request with chi route context and URL params set.
func requestWithChiURLParams(method, path string, body []byte, params map[string]string) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r
}

func TestAssetHandler_ListAssets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets ORDER BY id LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tags", "last_seen"}).
			AddRow(1, "asset1", "desc1", "{}", nil))

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	req := httptest.NewRequest("GET", "/assets?limit=10&offset=0", nil)
	rr := httptest.NewRecorder()
	h.ListAssets(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListAssets status: got %d, want 200", rr.Code)
	}
	var list []struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 1 || list[0].Name != "asset1" {
		t.Errorf("unexpected list: %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetHandler_GetAsset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets WHERE id=\$1`).
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tags", "last_seen"}).
			AddRow(1, "myasset", "mydesc", "{}", now))

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	req := requestWithChiURLParams("GET", "/assets/1", nil, map[string]string{"id": "1"})
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetAsset status: got %d, want 200", rr.Code)
	}
	var asset struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&asset); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if asset.ID != 1 || asset.Name != "myasset" || asset.Description != "mydesc" {
		t.Errorf("unexpected asset: %+v", asset)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetHandler_GetAsset_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, name, description, COALESCE\(tags, '{}'\), last_seen FROM assets WHERE id=\$1`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	req := requestWithChiURLParams("GET", "/assets/999", nil, map[string]string{"id": "999"})
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GetAsset status: got %d, want 404", rr.Code)
	}
	var out map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out["error"] != "asset not found" {
		t.Errorf("unexpected error body: %v", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetHandler_GetAsset_InvalidID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	req := requestWithChiURLParams("GET", "/assets/abc", nil, map[string]string{"id": "abc"})
	rr := httptest.NewRecorder()
	h.GetAsset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("GetAsset status: got %d, want 400", rr.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetHandler_CreateAsset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO assets \(name, description, tags\) VALUES \(\$1, \$2, \$3\) RETURNING id`).
		WithArgs("newasset", "newdesc", pq.Array([]string{})).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	body, _ := json.Marshal(map[string]string{"name": "newasset", "description": "newdesc"})
	req := httptest.NewRequest("POST", "/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateAsset(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("CreateAsset status: got %d, want 200", rr.Code)
	}
	var asset struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&asset); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if asset.ID != 10 || asset.Name != "newasset" || asset.Description != "newdesc" {
		t.Errorf("unexpected asset: %+v", asset)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestAssetHandler_CreateAsset_BadRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &AssetHandler{Repo: assetRepo}

	body, _ := json.Marshal(map[string]string{"name": "", "description": "ok"})
	req := httptest.NewRequest("POST", "/assets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateAsset(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("CreateAsset status: got %d, want 400", rr.Code)
	}
	var out struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Error != "validation failed" {
		t.Errorf("unexpected error: %v", out.Error)
	}
	if out.Fields["name"] != "required" {
		t.Errorf("unexpected fields: %v", out.Fields)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}
