package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/crucial707/hci-asset/internal/repo"
)

// ==========================
// Auth Handler
// ==========================
type AuthHandler struct {
	UserRepo *repo.UserRepo
}

// ==========================
// Register
// ==========================
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.Create(input.Username)
	if err != nil {
		JSONError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// ==========================
// Login (Stub)
// ==========================
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.GetByUsername(input.Username)
	if err != nil {
		JSONError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(user)
}
