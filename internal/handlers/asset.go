package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

const (
	MaxNameLength        = 100
	MaxDescriptionLength = 500
)

// ==========================
// Asset Input Struct
// ==========================
type AssetInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ==========================
// AssetHandler
// ==========================
type AssetHandler struct {
	Repo *repo.AssetRepo
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

	if input.Name == "" || input.Description == "" {
		JSONError(w, "name and description are required", http.StatusBadRequest)
		return
	}
	if len(input.Name) > MaxNameLength || len(input.Description) > MaxDescriptionLength {
		JSONError(w, "name or description too long", http.StatusBadRequest)
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
// List Assets
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
		assets, err = h.Repo.Search(search, limit, offset)
	} else {
		assets, err = h.Repo.List(limit, offset)
	}
	if err != nil {
		JSONError(w, "failed to fetch assets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// ==========================
// Get Asset
// ==========================
func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Get(id)
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

	if input.Name == "" || input.Description == "" {
		JSONError(w, "name and description required", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Update(id, input.Name, input.Description)
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

	if err := h.Repo.Delete(id); err != nil {
		JSONError(w, "failed to delete asset", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
