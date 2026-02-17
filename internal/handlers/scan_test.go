package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/crucial707/hci-asset/internal/repo"
)

func TestScanHandler_StartScan(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO scan_jobs .* RETURNING id`).
		WithArgs("192.168.1.0/24").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE scan_jobs SET status .* WHERE id`).
		WithArgs("error", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	// Use a nonexistent nmap so runScan fails quickly and does not call Repo
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "/nonexistent/nmap"}

	body, _ := json.Marshal(map[string]string{"target": "192.168.1.0/24"})
	req := httptest.NewRequest("POST", "/scans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.StartScan(rr, req)

	time.Sleep(50 * time.Millisecond) // allow runScan to persist
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
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "nmap"}

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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, status, started_at FROM scan_jobs ORDER BY id DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target", "status", "started_at"}))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM scan_jobs`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "nmap"}

	req := httptest.NewRequest("GET", "/scans", nil)
	rr := httptest.NewRecorder()
	h.ListScans(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("ListScans status: got %d, want 200", rr.Code)
	}
	var listResp struct {
		Items []struct {
			ID        int    `json:"id"`
			Target    string `json:"target"`
			Status    string `json:"status"`
			StartedAt string `json:"started_at"`
		} `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(listResp.Items) != 0 {
		t.Errorf("expected empty list, got %+v", listResp.Items)
	}
}

func TestScanHandler_GetScanStatus_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, target, status, started_at, completed_at, error, assets FROM scan_jobs WHERE id = \$1`).
		WithArgs(99).
		WillReturnError(sql.ErrNoRows)

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "nmap"}

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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO scan_jobs .* RETURNING id`).
		WithArgs("127.0.0.1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE scan_jobs SET status .* WHERE id`).
		WithArgs("error", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "/nonexistent/nmap"}

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
	time.Sleep(50 * time.Millisecond) // allow runScan to persist so mock expectations are met
}

func TestScanHandler_CancelScan_NotFound(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "nmap"}

	req := requestWithChiURLParams("POST", "/scans/99/cancel", []byte("{}"), map[string]string{"id": "99"})
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CancelScan(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("CancelScan status: got %d, want 404", rr.Code)
	}
}

func TestScanHandler_CancelScan(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`INSERT INTO scan_jobs .* RETURNING id`).
		WithArgs("10.0.0.1").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec(`UPDATE scan_jobs SET status .* WHERE id`).
		WithArgs("canceled", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	assetRepo := repo.NewAssetRepo(db)
	scanJobRepo := repo.NewScanJobRepo(db)
	h := &ScanHandler{Repo: assetRepo, ScanJobRepo: scanJobRepo, NmapPath: "/nonexistent/nmap"}

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
	time.Sleep(50 * time.Millisecond) // allow runScan to persist
}
