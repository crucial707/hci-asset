package handlers

import (
	"net/http"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
)

// ==========================
// UserHandler
// ==========================
type UserHandler struct {
	Repo *repo.UserRepo
}

// ==========================
// Create User (stub)
// ==========================
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	JSONError(w, "user creation not implemented", http.StatusNotImplemented)
}

// ==========================
// Get User
// ==========================
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id")
	JSONError(w, "get user not implemented", http.StatusNotImplemented)
}

// ==========================
// Update User
// ==========================
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id")
	JSONError(w, "update user not implemented", http.StatusNotImplemented)
}

// ==========================
// Delete User
// ==========================
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id")
	JSONError(w, "delete user not implemented", http.StatusNotImplemented)
}
