package handlers

import (
	"encoding/json"
	"net/http"
)

// ErrMessageInternal is the generic message for 500 responses. Do not expose internal details to clients.
const ErrMessageInternal = "internal server error"

// JSONError sends a JSON error response with a single "error" field.
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// JSONValidationError sends a JSON error response with "error" and optional "fields" for field-level details.
// status is typically http.StatusBadRequest (400).
func JSONValidationError(w http.ResponseWriter, message string, fields map[string]string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	out := map[string]interface{}{"error": message}
	if len(fields) > 0 {
		out["fields"] = fields
	}
	json.NewEncoder(w).Encode(out)
}
