package handlers

import (
	"encoding/json"
	"net/http"
)

// JSONError sends a JSON error response
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
