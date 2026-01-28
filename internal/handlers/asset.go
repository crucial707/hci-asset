package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type AssetHandler struct {
	Repo  *repo.AssetRepo
	Token string
}

//
// ==========================
// Middleware
// ==========================
//

func (h *AssetHandler) APITokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("X-API-Token")

		if token != h.Token {
			JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

//
// ==========================
// Create Asset
// ==========================
//

func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name" validate:"required,min=2,max=255"`
		Description string `json:"description" validate:"max=1000"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// ===== Validate input =====
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
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

//
// ==========================
// List Assets
// ==========================
//

func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	// Default pagination
	limit := 10
	offset := 0

	// Parse limit
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			limit = val
		}
	}

	// Parse offset
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	// Optional search by name
	name := r.URL.Query().Get("name")

	var assets []models.Asset
	var err error

	if name != "" {
		assets, err = h.Repo.SearchPaginated(name, limit, offset)
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

//
// ==========================
// Get Asset By ID
// ==========================
//

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

//
// ==========================
// Update Asset
// ==========================
//

func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid asset id", http.StatusBadRequest)
		return
	}

	var input struct {
		Name        string `json:"name" validate:"required,min=2,max=255"`
		Description string `json:"description" validate:"max=1000"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// ===== Validate input =====
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		JSONError(w, err.Error(), http.StatusBadRequest)
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

//
// ==========================
// Delete Asset
// ==========================
//

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
