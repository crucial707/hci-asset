package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ScheduleHandler handles scan schedule CRUD.
type ScheduleHandler struct {
	Repo *repo.ScheduleRepo
}

// ListSchedules returns paginated schedules (query: limit, offset).
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	list, err := h.Repo.List(r.Context(), limit, offset)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// GetSchedule returns one schedule by id.
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid schedule id", http.StatusBadRequest)
		return
	}

	s, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}
	if s == nil {
		JSONError(w, "schedule not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// CreateSchedule creates a new schedule. Body: {"target": "...", "cron_expr": "0 * * * *", "enabled": true}.
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Target   string `json:"target"`
		CronExpr string `json:"cron_expr"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	fields := make(map[string]string)
	if input.Target == "" {
		fields["target"] = "required"
	}
	if input.CronExpr == "" {
		fields["cron_expr"] = "required"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	s, err := h.Repo.Create(r.Context(), input.Target, input.CronExpr, enabled)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(s)
}

// UpdateSchedule updates a schedule. Body: {"target": "...", "cron_expr": "...", "enabled": true}.
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid schedule id", http.StatusBadRequest)
		return
	}

	var input struct {
		Target   string `json:"target"`
		CronExpr string `json:"cron_expr"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	fields := make(map[string]string)
	if input.Target == "" {
		fields["target"] = "required"
	}
	if input.CronExpr == "" {
		fields["cron_expr"] = "required"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	if err := h.Repo.Update(r.Context(), id, input.Target, input.CronExpr, enabled); err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	s, _ := h.Repo.GetByID(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// DeleteSchedule deletes a schedule.
func (h *ScheduleHandler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid schedule id", http.StatusBadRequest)
		return
	}

	if err := h.Repo.Delete(r.Context(), id); err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
