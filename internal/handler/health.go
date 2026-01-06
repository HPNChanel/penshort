package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthChecker defines an interface for checking service health.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// HealthHandler manages health check endpoints.
type HealthHandler struct {
	db    HealthChecker
	cache HealthChecker
}

// NewHealthHandler creates a new HealthHandler.
// Pass nil for db or cache if they are not yet initialized.
func NewHealthHandler(db, cache HealthChecker) *HealthHandler {
	return &HealthHandler{
		db:    db,
		cache: cache,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Healthz is a liveness probe endpoint.
// It returns 200 if the server is running.
// No dependency checks - this is for Kubernetes liveness probes.
//
// GET /healthz
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}
	writeJSON(w, http.StatusOK, response)
}

// Readyz is a readiness probe endpoint.
// It checks all dependencies and returns 200 only if all are healthy.
// For Kubernetes readiness probes - removes pod from LB if failing.
//
// GET /readyz
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	healthy := true

	// Check PostgreSQL
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["postgres"] = "error: " + err.Error()
			healthy = false
		} else {
			checks["postgres"] = "ok"
		}
	} else {
		checks["postgres"] = "not configured"
	}

	// Check Redis
	if h.cache != nil {
		if err := h.cache.Ping(ctx); err != nil {
			checks["redis"] = "error: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}
	} else {
		checks["redis"] = "not configured"
	}

	status := "ok"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status: status,
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
