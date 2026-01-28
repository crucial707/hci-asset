package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserRepo  *repo.UserRepo
	JWTSecret []byte
}

// ===== Register =====
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		JSONError(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.UserRepo.Create(input.Username, string(hash))
	if err != nil {
		JSONError(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ===== Login =====
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	user, err := h.UserRepo.GetByUsername(input.Username)
	if err != nil {
		JSONError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		JSONError(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	tokenString, err := token.SignedString(h.JWTSecret)
	if err != nil {
		JSONError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}
