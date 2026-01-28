package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
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
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
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

	assets, err := h.Repo.List()
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
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
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
