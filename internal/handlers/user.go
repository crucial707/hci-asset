package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	MaxUsernameLength = 50
	MaxPasswordLength = 100
)

// ==========================
// User Input Struct
// ==========================
type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ==========================
// UserHandler
// ==========================
type UserHandler struct {
	Repo *repo.UserRepo
}

// ==========================
// Create User
// ==========================
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Username == "" || input.Password == "" {
		JSONError(w, "username and password are required", http.StatusBadRequest)
		return
	}
	if len(input.Username) > MaxUsernameLength || len(input.Password) > MaxPasswordLength {
		JSONError(w, "username or password too long", http.StatusBadRequest)
		return
	}

	// Hash the password before storing
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		JSONError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.Repo.Create(input.Username, string(hashedPassword))
	if err != nil {
		JSONError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// Get User by ID
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

	var input UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if input.Username == "" || input.Password == "" {
		JSONError(w, "username and password required", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		JSONError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.Repo.Update(id, input.Username, string(hashedPassword))
	if err != nil {
		JSONError(w, "failed to update user", http.StatusInternalServerError)
		return
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

	w.WriteHeader(http.StatusNoContent)
}
