// Package handler provides HTTP request handlers.
package handler

import (
	"encoding/json"
	"net/http"
)

// Handler wraps application dependencies for HTTP handlers.
type Handler struct {
	// Dependencies will be added here in future phases
	// e.g., Repository, Cache, Logger
}

// New creates a new Handler instance.
func New() *Handler {
	return &Handler{}
}

// Hello is a simple hello endpoint for testing.
// GET /
func (h *Handler) Hello(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Hello from Penshort!",
		"version": "0.1.0",
	}
	writeJSON(w, http.StatusOK, response)
}

// NotFound handles 404 responses.
func (h *Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"error": "resource not found",
	}
	writeJSON(w, http.StatusNotFound, response)
}

// MethodNotAllowed handles 405 responses.
func (h *Handler) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"error": "method not allowed",
	}
	writeJSON(w, http.StatusMethodNotAllowed, response)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error in production, for now just ignore
		_ = err
	}
}
