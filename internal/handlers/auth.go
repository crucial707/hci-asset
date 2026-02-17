package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// ==========================
// Auth Handler
// ==========================
type AuthHandler struct {
	UserRepo     *repo.UserRepo
	Secret       []byte
	ExpireHours  int // JWT token lifetime in hours
}

// ==========================
// Register (role optional, default viewer; admin requires password)
// ==========================
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, "invalid json", http.StatusBadRequest)
		return
	}

	role := input.Role
	if role == "" {
		role = models.RoleViewer
	}
	fields := make(map[string]string)
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

	user, err := h.UserRepo.Create(r.Context(), input.Username, input.Password, role)
	if err != nil {
		// Idempotent: if user already exists, return existing user (200)
		if e, ok := err.(*pq.Error); ok && e.Code == "23505" {
			user, getErr := h.UserRepo.GetByUsername(r.Context(), input.Username)
			if getErr != nil {
				JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
			return
		}
		log.Printf("Register: create user failed: %v", err)
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// ==========================
// Login (only viewer can log in without password; admin and any non-viewer require password)
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

	user, err := h.UserRepo.GetByUsername(r.Context(), input.Username)
	if err != nil {
		JSONError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	// Only "viewer" (view only) can log in without a password; admin and any other role require password.
	if user.Role != models.RoleViewer {
		if input.Password == "" {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if user.PasswordHash == "" {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
			JSONError(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
	} else if user.PasswordHash != "" {
		// Viewer with optional password set: still allow username-only, but if password provided, verify it
		if input.Password != "" {
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
				JSONError(w, "invalid credentials", http.StatusUnauthorized)
				return
			}
		}
	}

	expHours := h.ExpireHours
	if expHours <= 0 {
		expHours = 24
	}
	// Create JWT token
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(time.Duration(expHours) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.Secret)
	if err != nil {
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": signed,
		"user":  user,
	})
}
