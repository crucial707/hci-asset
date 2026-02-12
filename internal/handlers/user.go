package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/middleware"
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
// Create User
// ==========================
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Username == "" {
		JSONError(w, "invalid JSON or missing username", http.StatusBadRequest)
		return
	}

	user, err := h.Repo.Create(input.Username)
	if err != nil {
		JSONError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(userID, "create", "user", user.ID, "")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// List Users
// ==========================
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.Repo.List()
	if err != nil {
		JSONError(w, "failed to fetch users", http.StatusInternalServerError)
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

	user, err := h.Repo.GetByID(id)
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

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Username == "" {
		JSONError(w, "invalid JSON or missing username", http.StatusBadRequest)
		return
	}

	user, err := h.Repo.Update(id, input.Username)
	if err != nil {
		JSONError(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(userID, "update", "user", id, "")
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

	if err := h.Repo.Delete(id); err != nil {
		JSONError(w, "failed to delete user", http.StatusInternalServerError)
		return
	}

	if h.AuditRepo != nil {
		if userID, ok := middleware.GetUserID(r.Context()); ok {
			_ = h.AuditRepo.Log(userID, "delete", "user", id, "")
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
