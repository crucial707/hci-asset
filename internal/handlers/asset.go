package handlers

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
)

// ==========================
// INPUT STRUCTS
// ==========================
type AssetInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ScanInput struct {
	Target string `json:"target"`
}

type AssetHandler struct {
	Repo *repo.AssetRepo
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
// CREATE ASSET
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
// LIST ASSETS (PAGINATION + SEARCH)
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
// GET ASSET BY ID
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
// UPDATE ASSET
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
// DELETE ASSET
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
// SCAN NETWORK HANDLER
// ==========================
func (h *AssetHandler) ScanNetwork(w http.ResponseWriter, r *http.Request) {
	var input ScanInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	target := strings.TrimSpace(input.Target)
	if target == "" {
		JSONError(w, "target is required", http.StatusBadRequest)
		return
	}

	// Run nmap
	cmd := exec.Command("C:\\Program Files (x86)\\Nmap\\nmap.exe", "-sn", target)
	out, err := cmd.Output()
	if err != nil {
		JSONError(w, "failed to run nmap: "+err.Error(), http.StatusInternalServerError)
		return
	}

	output := string(out)

	// Regex for host names and MAC addresses
	reIP := regexp.MustCompile(`Nmap scan report for (.+) \((\d+\.\d+\.\d+\.\d+)\)`)
	reMAC := regexp.MustCompile(`MAC Address: ([0-9A-F:]+) \((.+)\)`)

	var assets []models.Asset
	lines := strings.Split(output, "\n")
	var currentName, currentIP, currentMAC string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := reIP.FindStringSubmatch(line); matches != nil {
			currentName = matches[1]
			currentIP = matches[2]
		} else if matches := reMAC.FindStringSubmatch(line); matches != nil {
			currentMAC = matches[1]
		}

		if currentIP != "" && currentName != "" {
			desc := currentIP
			if currentMAC != "" {
				desc += " | " + currentMAC
			}
			asset, _ := h.Repo.Create(currentName, desc)
			assets = append(assets, asset)

			// Reset for next host
			currentName, currentIP, currentMAC = "", "", ""
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scanned_hosts": len(assets),
		"assets":        assets,
	})
}
