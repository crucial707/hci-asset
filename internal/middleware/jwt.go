package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type key string

const UserIDKey key = "user_id"
const RoleKey key = "role"

// GetUserID returns the user ID from the request context (set by JWTMiddleware). ok is false if not found.
func GetUserID(ctx context.Context) (userID int, ok bool) {
	v := ctx.Value(UserIDKey)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int)
	return id, ok
}

// GetRole returns the user role from the request context (set by JWTMiddleware). ok is false if not found.
func GetRole(ctx context.Context) (role string, ok bool) {
	v := ctx.Value(RoleKey)
	if v == nil {
		return "", false
	}
	role, ok = v.(string)
	return role, ok
}

func JWTMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return secret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				ctx := r.Context()
				ctx = context.WithValue(ctx, UserIDKey, int(claims["user_id"].(float64)))
				if role, ok := claims["role"].(string); ok {
					ctx = context.WithValue(ctx, RoleKey, role)
				} else {
					// Tokens issued before role existed: treat as viewer
					ctx = context.WithValue(ctx, RoleKey, models.RoleViewer)
				}
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}

// RequireAdmin returns 403 Forbidden if the request context role is not admin. Use after JWTMiddleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := GetRole(r.Context())
		if role != models.RoleAdmin {
			http.Error(w, "admin required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
