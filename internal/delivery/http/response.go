package http

import (
	"encoding/json"
	"net/http"
)

// Response standardizes API JSON output.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// RespondJSON sends a JSON response with status code.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: status >= 200 && status < 300,
		Data:    data,
	})
}

// RespondError sends a standardized JSON error response.
func RespondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	})
}
