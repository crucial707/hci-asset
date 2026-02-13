package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ==========================
// Auth Handler
// ==========================
type AuthHandler struct {
	UserRepo *repo.UserRepo
	Secret   []byte
}

// ==========================
// Register (optional password; stored as bcrypt hash)
// ==========================
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.Create(input.Username, input.Password)
	if err != nil {
		// Idempotent: if user already exists, return existing user (200)
		if e, ok := err.(*pq.Error); ok && e.Code == "23505" {
			user, getErr := h.UserRepo.GetByUsername(input.Username)
			if getErr != nil {
				JSONError(w, "failed to create user", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
		log.Printf("Register: create user failed: %v", err)
		JSONError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// Login (username required; if user has password set, password required and verified)
// ==========================
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
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

	if user.PasswordHash != "" {
		if input.Password == "" {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
	}

	// Create JWT token
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.Secret)
	if err != nil {
		JSONError(w, "failed to issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": signed,
		"user":  user,
	})
}
