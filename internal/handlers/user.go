package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/middleware"
	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ==========================
// UserHandler
// ==========================
type UserHandler struct {
	Repo      *repo.UserRepo
	AuditRepo *repo.AuditRepo
}

// ==========================
// Create User (optional role default viewer; admin requires password)
// ==========================
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	fields := make(map[string]string)
	if input.Username == "" {
		fields["username"] = "required"
	}
	role := input.Role
	if role == "" {
		role = models.RoleViewer
	}
	if role != models.RoleViewer && role != models.RoleAdmin {
		fields["role"] = "must be viewer or admin"
	}
	if role == models.RoleAdmin && input.Password == "" {
		fields["password"] = "required for admin"
	}
	if len(fields) > 0 {
		JSONValidationError(w, "validation failed", fields, http.StatusBadRequest)
		return
	}

	user, err := h.Repo.Create(r.Context(), input.Username, input.Password, role)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "create", "user", user.ID, "")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// List Users
// ==========================
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Repo.List(r.Context())
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// ==========================
// Get User
// ==========================
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	user, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		JSONError(w, "user not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// Update User
// ==========================
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	var input struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if input.Username == "" {
		JSONValidationError(w, "validation failed", map[string]string{"username": "required"}, http.StatusBadRequest)
		return
	}

	user, err := h.Repo.Update(r.Context(), id, input.Username)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "update", "user", id, "")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// Delete User
// ==========================
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		JSONError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	if err := h.Repo.Delete(r.Context(), id); err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(r.Context(), userID, "delete", "user", id, "")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
