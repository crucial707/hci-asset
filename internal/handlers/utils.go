package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse defines standard error payload
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
	})
}
