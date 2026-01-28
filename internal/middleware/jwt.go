package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type key string

const UserIDKey key = "user_id"

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
				ctx := context.WithValue(r.Context(), UserIDKey, int(claims["user_id"].(float64)))
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}
