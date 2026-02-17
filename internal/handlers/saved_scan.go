package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// SavedScanHandler handles saved scan CRUD and run.
type SavedScanHandler struct {
	Repo        *repo.SavedScanRepo
	ScanHandler *ScanHandler // used to start a scan from a saved target
}

// ListSavedScans returns all saved scans.
func (h *SavedScanHandler) ListSavedScans(w http.ResponseWriter, r *http.Request) {
	list, err := h.Repo.List(r.Context())
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"items": list})
}

// GetSavedScan returns one saved scan by id.
func (h *SavedScanHandler) GetSavedScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
		return
	}
	saved, err := h.Repo.GetByID(r.Context(), id)
	if err != nil || saved == nil {
		JSONError(w, "saved scan not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

// CreateSavedScan creates a new saved scan (name + target).
func (h *SavedScanHandler) CreateSavedScan(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name   string `json:"name"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	fields := make(map[string]string)
	if input.Name == "" {
		fields["name"] = "required"
	}
	if input.Target == "" {
		fields["target"] = "required"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}
	saved, err := h.Repo.Create(r.Context(), input.Name, input.Target)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

// UpdateSavedScan updates a saved scan.
func (h *SavedScanHandler) UpdateSavedScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var input struct {
		Name   string `json:"name"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	fields := make(map[string]string)
	if input.Name == "" {
		fields["name"] = "required"
	}
	if input.Target == "" {
		fields["target"] = "required"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}
	saved, err := h.Repo.Update(r.Context(), id, input.Name, input.Target)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	if saved == nil {
		JSONError(w, "saved scan not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

// DeleteSavedScan deletes a saved scan.
func (h *SavedScanHandler) DeleteSavedScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := h.Repo.Delete(r.Context(), id); err != nil {
		if err.Error() == "saved scan not found" {
			JSONError(w, "saved scan not found", http.StatusNotFound)
			return
		}
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RunSavedScan starts a scan using the saved scan's target and returns the new job ID.
func (h *SavedScanHandler) RunSavedScan(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid id", http.StatusBadRequest)
		return
	}
	saved, err := h.Repo.GetByID(r.Context(), id)
	if err != nil || saved == nil {
		JSONError(w, "saved scan not found", http.StatusNotFound)
		return
	}
	jobID := h.ScanHandler.StartScanTarget(r.Context(), saved.Target)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"job_id": jobID,
		"status": "running",
	})
}
