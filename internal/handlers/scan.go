package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/crucial707/hci-asset/internal/metrics"
	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ==========================
// ScanJob Struct
// ==========================
type ScanJob struct {
	Target      string         `json:"target"`
	Status      string         `json:"status"` // running, complete, canceled, error
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Assets      []models.Asset `json:"assets,omitempty"`
	Error       string         `json:"error,omitempty"`
	cancel      chan struct{}  `json:"-"`
}

// ==========================
// ScanHandler
// ==========================
type ScanHandler struct {
	Repo       *repo.AssetRepo
	ScanJobRepo *repo.ScanJobRepo
	NmapPath   string // path to nmap executable (e.g. "nmap" or "C:\\Program Files (x86)\\Nmap\\nmap.exe")
	scanJobs   map[string]*ScanJob // in-memory only for running jobs (for cancel channel)
	scanJobsMu sync.Mutex
}

// ==========================
// Start Scan
// ==========================
func (h *ScanHandler) StartScan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Target == "" {
		JSONError(w, "invalid JSON or missing target", http.StatusBadRequest)
		return
	}

	jobID := h.StartScanTarget(r.Context(), input.Target)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "running",
	})
}

// StartScanTarget starts a scan for the given target and returns the job ID.
// Used by the API (StartScan) and by the schedule runner. Persists the job to DB.
func (h *ScanHandler) StartScanTarget(ctx context.Context, target string) string {
	id, err := h.ScanJobRepo.Create(ctx, target)
	if err != nil {
		// Fallback to in-memory-only id if DB fails (e.g. table missing)
		h.scanJobsMu.Lock()
		if h.scanJobs == nil {
			h.scanJobs = make(map[string]*ScanJob)
		}
		jobID := strconv.Itoa(len(h.scanJobs)+1) + "-mem"
		job := &ScanJob{Target: target, Status: "running", StartedAt: time.Now(), cancel: make(chan struct{})}
		h.scanJobs[jobID] = job
		h.scanJobsMu.Unlock()
		metrics.IncScanJobsRunning()
		go h.runScan(jobID, target, job.cancel)
		return jobID
	}
	jobID := strconv.Itoa(id)
	job := &ScanJob{
		Target:    target,
		Status:    "running",
		StartedAt: time.Now(),
		cancel:    make(chan struct{}),
	}
	h.scanJobsMu.Lock()
	if h.scanJobs == nil {
		h.scanJobs = make(map[string]*ScanJob)
	}
	h.scanJobs[jobID] = job
	h.scanJobsMu.Unlock()

	metrics.IncScanJobsRunning()
	go h.runScan(jobID, target, job.cancel)
	return jobID
}

// ==========================
// List Scans (recent job IDs with target, status, started_at) from DB.
// ==========================
func (h *ScanHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}
	list, err := h.ScanJobRepo.List(r.Context(), limit, offset)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	total, err := h.ScanJobRepo.Count(r.Context())
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":  list,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ==========================
// Get Scan Status (from memory if running, else from DB).
// ==========================
func (h *ScanHandler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	h.scanJobsMu.Lock()
	job, inMem := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	if inMem {
		// Return in-memory job with id for consistency
		out := map[string]interface{}{
			"id":         jobID,
			"target":    job.Target,
			"status":    job.Status,
			"started_at": job.StartedAt,
			"assets":    job.Assets,
			"error":     job.Error,
		}
		if job.CompletedAt != nil {
			out["completed_at"] = job.CompletedAt
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
		return
	}

	id, err := strconv.Atoi(jobID)
	if err != nil {
		JSONError(w, "scan job not found", http.StatusNotFound)
		return
	}
	row, err := h.ScanJobRepo.GetByID(r.Context(), id)
	if err != nil || row == nil {
		JSONError(w, "scan job not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(row)
}

// ClearScans deletes all scan job records from the DB so the active scans list starts fresh.
// Running in-memory jobs are not stopped; they will complete and no longer appear after clear.
func (h *ScanHandler) ClearScans(w http.ResponseWriter, r *http.Request) {
	if err := h.ScanJobRepo.DeleteAll(r.Context()); err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ==========================
// Cancel Scan (only works for running jobs in memory).
// ==========================
func (h *ScanHandler) CancelScan(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	h.scanJobsMu.Lock()
	job, exists := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	if !exists {
		JSONError(w, "scan job not found or not running", http.StatusNotFound)
		return
	}

	select {
	case <-job.cancel:
	default:
		close(job.cancel)
		job.Status = "canceled"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": jobID, "target": job.Target, "status": job.Status,
		"started_at": job.StartedAt, "completed_at": job.CompletedAt,
		"assets": job.Assets, "error": job.Error,
	})
}

// ==========================
// Internal Scan Executor (persists result to DB when jobID is numeric).
// ==========================
func (h *ScanHandler) runScan(jobID, target string, cancelCh chan struct{}) {
	ctx := context.Background()
	h.scanJobsMu.Lock()
	job := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	defer func() {
		metrics.DecScanJobsRunning()
		metrics.IncScanJobsTotal(job.Status)
	}()

	persist := func() {
		id, err := strconv.Atoi(jobID)
		if err != nil {
			return // fallback in-memory job (e.g. "3-mem"), skip persist
		}
		_ = h.ScanJobRepo.Update(ctx, id, job.Status, job.CompletedAt, job.Error, job.Assets)
	}

	nmapExe := h.NmapPath
	if nmapExe == "" {
		nmapExe = "nmap"
	}
	// Use grepable output (-oG -) so we can parse only hosts with Status: Up.
	// Default interactive output lists "Nmap scan report" for every IP; we only want hosts that responded.
	cmd := exec.Command(nmapExe, "-sn", "-oG", "-", target)
	outputBytes, err := cmd.CombinedOutput()
	now := time.Now()
	job.CompletedAt = &now
	if err != nil {
		job.Status = "error"
		job.Error = err.Error()
		persist()
		return
	}

	output := string(outputBytes)
	lines := strings.Split(output, "\n")
	// Only consider lines that indicate the host is up (grepable: "Host: IP (name)	Status: Up")
	// Hostname may be empty: "Host: 192.168.1.1 ()	Status: Up"
	reUp := regexp.MustCompile(`Host:\s+([\d.]+)\s*\(([^)]*)\)\s+Status:\s*Up\b`)

	var discovered []models.Asset
	for _, line := range lines {
		select {
		case <-cancelCh:
			done := time.Now()
			job.CompletedAt = &done
			job.Status = "canceled"
			persist()
			return
		default:
		}

		match := reUp.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		ip := match[1]
		hostname := strings.TrimSpace(match[2])
		displayName := hostname
		if displayName == "" {
			displayName = ip
		}

		desc := "Discovered device"
		asset, err := h.Repo.UpsertDiscovered(ctx, displayName, desc)
		if err != nil {
			if job.Error == "" {
				job.Error = "one or more assets failed to upsert"
			}
			continue
		}
		asset.NetworkName = ip
		discovered = append(discovered, *asset)
	}

	job.Assets = discovered
	if job.Status != "canceled" {
		done := time.Now()
		job.CompletedAt = &done
		job.Status = "complete"
	}
	persist()
}
