package httputil

import (
	"encoding/json"
	"net/http"
)

// JSONError writes a JSON error response with the correct Content-Type.
// Use this instead of http.Error to ensure the frontend can parse error messages.
func JSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
