package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ========================
// HANDLER STRUCT
// ========================

type AssetHandler struct {
	Repo  *repo.AssetRepo
	Token string
}

// ========================
// UTILS: JSON ERROR RESPONSE
// ========================

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// ========================
// MIDDLEWARE: API TOKEN AUTH
// ========================

// Exported so main.go can use it
func (h *AssetHandler) APITokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqToken := r.Header.Get("X-API-Token")
		if reqToken != h.Token {
			JSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// ========================
// CREATE ASSET
// ========================

func (h *AssetHandler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		JSONError(w, "method not allowed", http.StatusMethodNotAllowed)
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

	asset, err := h.Repo.Create(input.Name, input.Description)
	if err != nil {
		JSONError(w, "failed to create asset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset)
}

// ========================
// LIST ALL ASSETS
// ========================

func (h *AssetHandler) ListAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.Repo.List()
	if err != nil {
		JSONError(w, "failed to fetch assets", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assets)
}

// ========================
// GET SINGLE ASSET
// ========================

func (h *AssetHandler) GetAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
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

// ========================
// DELETE ASSET
// ========================

func (h *AssetHandler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.Repo.DeleteByID(id); err != nil {
		JSONError(w, "failed to delete asset", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ========================
// UPDATE ASSET
// ========================

func (h *AssetHandler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
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

	updatedAsset, err := h.Repo.UpdateByID(id, input.Name, input.Description)
	if err != nil {
		JSONError(w, "failed to update asset", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedAsset)
}
