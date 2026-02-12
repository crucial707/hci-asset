package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/repo"
)

func TestScanHandler_StartScan(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	// Use a nonexistent nmap so runScan fails quickly and does not call Repo
	h := &ScanHandler{Repo: assetRepo, NmapPath: "/nonexistent/nmap"}

	body, _ := json.Marshal(map[string]string{"target": "192.168.1.0/24"})
	req := httptest.NewRequest("POST", "/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.StartScan(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("StartScan status: got %d, want 200", rr.Code)
	}
	var out struct {
		JobID  string `json:"job_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.JobID == "" || out.Status != "running" {
		t.Errorf("unexpected response: %+v", out)
	}
}

func TestScanHandler_StartScan_BadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "nmap"}

	body, _ := json.Marshal(map[string]string{"target": ""})
	req := httptest.NewRequest("POST", "/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.StartScan(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("StartScan status: got %d, want 400", rr.Code)
	}
}

func TestScanHandler_ListScans(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "nmap"}

	req := httptest.NewRequest("GET", "/scans", nil)
	rr := httptest.NewRecorder()
	h.ListScans(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListScans status: got %d, want 200", rr.Code)
	}
	var list []struct {
		ID        string `json:"id"`
		Target    string `json:"target"`
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %+v", list)
	}
}

func TestScanHandler_GetScanStatus_NotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "nmap"}

	req := requestWithChiURLParams("GET", "/scans/99", nil, map[string]string{"id": "99"})
	rr := httptest.NewRecorder()
	h.GetScanStatus(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("GetScanStatus status: got %d, want 404", rr.Code)
	}
	var out map[string]string
	_ = json.NewDecoder(rr.Body).Decode(&out)
	if out["error"] != "scan job not found" {
		t.Errorf("unexpected error body: %v", out)
	}
}

func TestScanHandler_StartScanThenGetStatus(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "/nonexistent/nmap"}

	body, _ := json.Marshal(map[string]string{"target": "127.0.0.1"})
	req := httptest.NewRequest("POST", "/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.StartScan(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("StartScan: got status %d", rr.Code)
	}
	var startOut struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&startOut); err != nil || startOut.JobID == "" {
		t.Fatalf("StartScan response: %v", err)
	}

	req2 := requestWithChiURLParams("GET", "/scans/"+startOut.JobID, nil, map[string]string{"id": startOut.JobID})
	rr2 := httptest.NewRecorder()
	h.GetScanStatus(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("GetScanStatus status: got %d, want 200", rr2.Code)
	}
	var job struct {
		Target    string `json:"target"`
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
	}
	if err := json.NewDecoder(rr2.Body).Decode(&job); err != nil {
		t.Fatalf("decode GetScanStatus response: %v", err)
	}
	if job.Target != "127.0.0.1" {
		t.Errorf("GetScanStatus job target: got %q", job.Target)
	}
}

func TestScanHandler_CancelScan_NotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "nmap"}

	req := requestWithChiURLParams("POST", "/scans/99/cancel", []byte("{}"), map[string]string{"id": "99"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CancelScan(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("CancelScan status: got %d, want 404", rr.Code)
	}
}

func TestScanHandler_CancelScan(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	h := &ScanHandler{Repo: assetRepo, NmapPath: "/nonexistent/nmap"}

	body, _ := json.Marshal(map[string]string{"target": "10.0.0.1"})
	req := httptest.NewRequest("POST", "/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.StartScan(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("StartScan: got status %d", rr.Code)
	}
	var startOut struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&startOut); err != nil || startOut.JobID == "" {
		t.Fatalf("StartScan response: %v", err)
	}

	cancelReq := requestWithChiURLParams("POST", "/scans/"+startOut.JobID+"/cancel", []byte("{}"), map[string]string{"id": startOut.JobID})
	cancelReq.Header.Set("Content-Type", "application/json")
	rrCancel := httptest.NewRecorder()
	h.CancelScan(rrCancel, cancelReq)

	if rrCancel.Code != http.StatusOK {
		t.Errorf("CancelScan status: got %d, want 200", rrCancel.Code)
	}
	var job struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rrCancel.Body).Decode(&job); err != nil {
		t.Fatalf("decode CancelScan response: %v", err)
	}
	if job.Status != "canceled" {
		t.Errorf("CancelScan job status: got %q, want canceled", job.Status)
	}
}
