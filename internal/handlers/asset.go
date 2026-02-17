package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/middleware"
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
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// ==========================
// AssetHandler
// ==========================
type AssetHandler struct {
	Repo      *repo.AssetRepo
	AuditRepo *repo.AuditRepo
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

	fields := make(map[string]string)
	if input.Name == "" {
		fields["name"] = "required"
	}
	if input.Description == "" {
		fields["description"] = "required"
	}
	if len(input.Name) > MaxNameLength {
		fields["name"] = "too long"
	}
	if len(input.Description) > MaxDescriptionLength {
		fields["description"] = "too long"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Create(r.Context(), input.Name, input.Description, input.Tags)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "create", "asset", asset.ID, "")
		}
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
	tag := r.URL.Query().Get("tag")

	var assets []models.Asset
	var total int
	var err error
	switch {
	case tag != "":
		assets, err = h.Repo.ListByTag(r.Context(), tag, limit, offset)
		if err == nil {
			total, err = h.Repo.CountByTag(r.Context(), tag)
		}
	case search != "":
		assets, err = h.Repo.Search(r.Context(), search, limit, offset)
		if err == nil {
			total, err = h.Repo.CountSearch(r.Context(), search)
		}
	default:
		assets, err = h.Repo.List(r.Context(), limit, offset)
		if err == nil {
			total, err = h.Repo.Count(r.Context())
		}
	}
	if err != nil {
		log.Printf("ListAssets error: %v", err)
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":  assets,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
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

	asset, err := h.Repo.Get(r.Context(), id)
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

	fields := make(map[string]string)
	if input.Name == "" {
		fields["name"] = "required"
	}
	if input.Description == "" {
		fields["description"] = "required"
	}
	if len(input.Name) > MaxNameLength {
		fields["name"] = "too long"
	}
	if len(input.Description) > MaxDescriptionLength {
		fields["description"] = "too long"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Update(r.Context(), id, input.Name, input.Description, input.Tags)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "update", "asset", id, "")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// ==========================
// Heartbeat updates last_seen for an asset (agent check-in).
// ==========================
func (h *AssetHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	asset, err := h.Repo.Heartbeat(r.Context(), id)
	if err != nil {
		if err.Error() == "asset not found" {
			JSONError(w, "asset not found", http.StatusNotFound)
			return
		}
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
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

	if err := h.Repo.Delete(r.Context(), id); err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "delete", "asset", id, "")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
