package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
	NmapPath   string // path to nmap executable (e.g. "nmap" or "C:\\Program Files (x86)\\Nmap\\nmap.exe")
	scanJobs   map[string]*ScanJob
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

	jobID := h.StartScanTarget(input.Target)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "running",
	})
}

// StartScanTarget starts a scan for the given target and returns the job ID.
// Used by the API (StartScan) and by the schedule runner.
func (h *ScanHandler) StartScanTarget(target string) string {
	h.scanJobsMu.Lock()
	if h.scanJobs == nil {
		h.scanJobs = make(map[string]*ScanJob)
	}
	jobID := strconv.Itoa(len(h.scanJobs) + 1)
	job := &ScanJob{
		Target:    target,
		Status:    "running",
		StartedAt: time.Now(),
		cancel:    make(chan struct{}),
	}
	h.scanJobs[jobID] = job
	h.scanJobsMu.Unlock()

	go h.runScan(jobID, target, job.cancel)
	return jobID
}

// ==========================
// List Scans (recent job IDs with target, status, started_at)
// ==========================
func (h *ScanHandler) ListScans(w http.ResponseWriter, r *http.Request) {
	h.scanJobsMu.Lock()
	defer h.scanJobsMu.Unlock()

	type jobSummary struct {
		ID        string    `json:"id"`
		Target    string    `json:"target"`
		Status    string    `json:"status"`
		StartedAt time.Time `json:"started_at"`
	}

	var ids []int
	for id := range h.scanJobs {
		n, err := strconv.Atoi(id)
		if err != nil {
			continue
		}
		ids = append(ids, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))

	const maxRecent = 20
	var list []jobSummary
	for i, n := range ids {
		if i >= maxRecent {
			break
		}
		idStr := strconv.Itoa(n)
		job, ok := h.scanJobs[idStr]
		if !ok {
			continue
		}
		list = append(list, jobSummary{
			ID:        idStr,
			Target:    job.Target,
			Status:    job.Status,
			StartedAt: job.StartedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// ==========================
// Get Scan Status
// ==========================
func (h *ScanHandler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	h.scanJobsMu.Lock()
	job, exists := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	if !exists {
		JSONError(w, "scan job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// ==========================
// Cancel Scan
// ==========================
func (h *ScanHandler) CancelScan(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "id")
	h.scanJobsMu.Lock()
	job, exists := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	if !exists {
		JSONError(w, "scan job not found", http.StatusNotFound)
		return
	}

	select {
	case <-job.cancel:
	default:
		close(job.cancel)
		job.Status = "canceled"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// ==========================
// Internal Scan Executor
// ==========================
func (h *ScanHandler) runScan(jobID, target string, cancelCh chan struct{}) {
	h.scanJobsMu.Lock()
	job := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	nmapExe := h.NmapPath
	if nmapExe == "" {
		nmapExe = "nmap"
	}
	cmd := exec.Command(nmapExe, "-sn", target)
	outputBytes, err := cmd.CombinedOutput()
	now := time.Now()
	job.CompletedAt = &now
	if err != nil {
		job.Status = "error"
		job.Error = err.Error()
		return
	}

	output := string(outputBytes)
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`Nmap scan report for (.+) \(([\d\.]+)\)|Nmap scan report for ([\d\.]+)`)

	var discovered []models.Asset
	for _, line := range lines {
		select {
		case <-cancelCh:
			done := time.Now()
			job.CompletedAt = &done
			job.Status = "canceled"
			return
		default:
		}

		match := re.FindStringSubmatch(line)
		if match != nil {
			var name, ip string
			if match[2] != "" {
				name = match[1]
				ip = match[2]
			} else if match[3] != "" {
				name = ""
				ip = match[3]
			} else {
				continue
			}

			desc := "Discovered device"
			displayName := name
			if displayName == "" {
				displayName = ip
			}

			asset, err := h.Repo.UpsertDiscovered(displayName, desc)
			if err != nil {
				// Best-effort: log in job error but continue scanning others
				if job.Error == "" {
					job.Error = "one or more assets failed to upsert"
				}
				continue
			}

			// Attach network info in-memory for the response
			asset.NetworkName = ip

			discovered = append(discovered, *asset)
		}
	}

	job.Assets = discovered
	if job.Status != "canceled" {
		done := time.Now()
		job.CompletedAt = &done
		job.Status = "complete"
	}
}
