package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockHealthChecker is a mock implementation of HealthChecker for testing.
type mockHealthChecker struct {
	err error
}

func (m *mockHealthChecker) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthHandler_Healthz(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", response.Status)
	}
}

func TestHealthHandler_Readyz_AllHealthy(t *testing.T) {
	db := &mockHealthChecker{}
	cache := &mockHealthChecker{}
	h := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got %s", response.Status)
	}

	if response.Checks["postgres"] != "ok" {
		t.Errorf("expected postgres check 'ok', got %s", response.Checks["postgres"])
	}

	if response.Checks["redis"] != "ok" {
		t.Errorf("expected redis check 'ok', got %s", response.Checks["redis"])
	}
}

func TestHealthHandler_Readyz_DatabaseUnhealthy(t *testing.T) {
	db := &mockHealthChecker{err: errors.New("connection refused")}
	cache := &mockHealthChecker{}
	h := NewHealthHandler(db, cache)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rec.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %s", response.Status)
	}

	if response.Checks["postgres"] != "error: connection refused" {
		t.Errorf("unexpected postgres check: %s", response.Checks["postgres"])
	}
}

func TestHealthHandler_Readyz_NoDependencies(t *testing.T) {
	h := NewHealthHandler(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	h.Readyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Checks["postgres"] != "not configured" {
		t.Errorf("expected 'not configured', got %s", response.Checks["postgres"])
	}
}
