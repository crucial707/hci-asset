package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
)

// AssetInput defines the structure for creating/updating an asset
type AssetInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AssetHandler holds repositories and scan state
type AssetHandler struct {
	Repo *repo.AssetRepo

	// scanJobs tracks ongoing and completed scans
	scanJobs   map[string]*ScanJob
	scanJobsMu sync.Mutex
}

// ScanJob represents a single network scan
type ScanJob struct {
	Target string         `json:"target"`
	Status string         `json:"status"` // running, complete, error
	Assets []models.Asset `json:"assets"`
	Error  string         `json:"error,omitempty"`
}

// ==========================
// JSON Error Helper
// ==========================
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// ==========================
// Create Asset
// ==========================
func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var input AssetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Description == "" {
		JSONError(w, "description is required", http.StatusBadRequest)
		return
	}
	if len(input.Name) > MaxNameLength {
		JSONError(w, "name cannot exceed 100 characters", http.StatusBadRequest)
		return
	}
	if len(input.Description) > MaxDescriptionLength {
		JSONError(w, "description cannot exceed 500 characters", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Create(input.Name, input.Description)
	if err != nil {
		JSONError(w, "failed to create asset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// ==========================
// List Assets (Pagination + Search)
// ==========================
func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	limit := 10
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

	search := r.URL.Query().Get("search")

	var assets []models.Asset
	var err error

	if search != "" {
		assets, err = h.Repo.SearchPaginated(search, limit, offset)
	} else {
		assets, err = h.Repo.ListPaginated(limit, offset)
	}

	if err != nil {
		JSONError(w, "failed to fetch assets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// ==========================
// Get Asset By ID
// ==========================
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.GetByID(id)
	if err != nil {
		JSONError(w, "asset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// ==========================
// Update Asset
// ==========================
func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	var input AssetInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Name == "" {
		JSONError(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Description == "" {
		JSONError(w, "description is required", http.StatusBadRequest)
		return
	}
	if len(input.Name) > MaxNameLength {
		JSONError(w, "name cannot exceed 100 characters", http.StatusBadRequest)
		return
	}
	if len(input.Description) > MaxDescriptionLength {
		JSONError(w, "description cannot exceed 500 characters", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.UpdateByID(id, input.Name, input.Description)
	if err != nil {
		JSONError(w, "failed to update asset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// ==========================
// Delete Asset
// ==========================
func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	if err := h.Repo.DeleteByID(id); err != nil {
		JSONError(w, "failed to delete asset", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ==========================
// Scan Network Handler (single, integrated)
// ==========================
func (h *AssetHandler) ScanNetwork(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var body struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Target == "" {
		JSONError(w, "target is required", http.StatusBadRequest)
		return
	}

	// Initialize scanJobs map if nil
	h.scanJobsMu.Lock()
	if h.scanJobs == nil {
		h.scanJobs = make(map[string]*ScanJob)
	}
	// Create job
	jobID := strconv.FormatInt(int64(len(h.scanJobs)+1), 10)
	job := &ScanJob{
		Target: body.Target,
		Status: "running",
	}
	h.scanJobs[jobID] = job
	h.scanJobsMu.Unlock()

	// Run scan async
	go h.runScan(jobID, body.Target)

	// Return job ID immediately
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "running",
	})
}

// ==========================
// Get Scan Status
// ==========================
func (h *AssetHandler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
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
// Internal scan executor
// ==========================
func (h *AssetHandler) runScan(jobID, target string) {
	h.scanJobsMu.Lock()
	job := h.scanJobs[jobID]
	h.scanJobsMu.Unlock()

	cmd := exec.Command("C:\\Program Files (x86)\\Nmap\\nmap.exe", "-sn", target)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		job.Status = "error"
		job.Error = err.Error()
		return
	}

	output := string(outputBytes)
	re := regexp.MustCompile(`Nmap scan report for (.+) \(([\d\.]+)\)|Nmap scan report for ([\d\.]+)`)
	lines := strings.Split(output, "\n")

	var discovered []models.Asset
	for _, line := range lines {
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
			asset, _ := h.Repo.Create(name+" ("+ip+")", desc)
			discovered = append(discovered, asset)
		}
	}

	job.Assets = discovered
	job.Status = "complete"
}
