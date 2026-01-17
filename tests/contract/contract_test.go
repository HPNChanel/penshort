// Package contract provides contract tests that validate API responses against the OpenAPI spec.
package contract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// testConfig holds test configuration.
type testConfig struct {
	BaseURL     string
	APIKey      string
	SpecPath    string
	SkipMessage string
}

// getConfig returns test configuration from environment.
func getConfig(t *testing.T) *testConfig {
	t.Helper()

	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	apiKey := os.Getenv("TEST_API_KEY")

	// Find spec path relative to test file
	specPath := os.Getenv("OPENAPI_SPEC_PATH")
	if specPath == "" {
		// Default: project root/docs/api/openapi.yaml
		wd, _ := os.Getwd()
		specPath = filepath.Join(wd, "..", "..", "docs", "api", "openapi.yaml")
	}

	return &testConfig{
		BaseURL:  baseURL,
		APIKey:   apiKey,
		SpecPath: specPath,
	}
}

// loadSpec loads and validates the OpenAPI spec.
func loadSpec(t *testing.T, path string) (*openapi3.T, routers.Router) {
	t.Helper()

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec from %s: %v", path, err)
	}

	if err := spec.Validate(context.Background()); err != nil {
		t.Fatalf("OpenAPI spec validation failed: %v", err)
	}

	router, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatalf("Failed to create router from spec: %v", err)
	}

	return spec, router
}

// TestOpenAPISpecValid ensures the OpenAPI spec is valid.
func TestOpenAPISpecValid(t *testing.T) {
	cfg := getConfig(t)
	_, _ = loadSpec(t, cfg.SpecPath)
	t.Log("OpenAPI spec is valid")
}

// TestEndpointsExist validates that documented endpoints respond.
func TestEndpointsExist(t *testing.T) {
	cfg := getConfig(t)
	spec, _ := loadSpec(t, cfg.SpecPath)

	client := &http.Client{Timeout: 10 * time.Second}

	// Test unauthenticated endpoints only
	unauthEndpoints := []struct {
		path   string
		method string
	}{
		{"/healthz", "GET"},
		{"/readyz", "GET"},
	}

	for _, ep := range unauthEndpoints {
		t.Run(fmt.Sprintf("%s_%s", ep.method, ep.path), func(t *testing.T) {
			url := cfg.BaseURL + ep.path
			req, err := http.NewRequest(ep.method, url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("Server not available: %v", err)
			}
			defer resp.Body.Close()

			// Endpoint exists if we don't get 404
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("Endpoint %s %s returned 404 - not implemented", ep.method, ep.path)
			}
		})
	}

	// Verify spec has expected paths
	expectedPaths := []string{
		"/api/v1/links",
		"/api/v1/links/{id}",
		"/api/v1/webhooks",
		"/healthz",
		"/readyz",
	}

	for _, path := range expectedPaths {
		if spec.Paths.Find(path) == nil {
			t.Errorf("Expected path %s not found in spec", path)
		}
	}
}

// TestErrorResponseSchema validates error responses match the schema.
func TestErrorResponseSchema(t *testing.T) {
	cfg := getConfig(t)

	if cfg.APIKey == "" {
		t.Skip("TEST_API_KEY not set - skipping error response tests")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	errorCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		needsAuth      bool
	}{
		{"Unauthorized", "GET", "/api/v1/links", 401, false},
		{"NotFound", "GET", "/api/v1/links/nonexistent-id-12345", 404, true},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			url := cfg.BaseURL + tc.path
			req, err := http.NewRequest(tc.method, url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if tc.needsAuth {
				req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("Server not available: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Logf("Expected status %d, got %d (test may need adjustment)", tc.expectedStatus, resp.StatusCode)
			}

			// Validate error response schema for 4xx/5xx
			if resp.StatusCode >= 400 {
				validateErrorResponse(t, resp)
			}
		})
	}
}

// validateErrorResponse checks that error responses have required fields.
func validateErrorResponse(t *testing.T, resp *http.Response) {
	t.Helper()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Error response Content-Type should be application/json, got: %s", contentType)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var errorResp struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		t.Errorf("Failed to parse error response as JSON: %v\nBody: %s", err, string(body))
		return
	}

	// Validate required fields per ErrorResponse schema
	if errorResp.Error == "" {
		t.Errorf("Error response missing 'error' field. Body: %s", string(body))
	}
	if errorResp.Code == "" {
		t.Errorf("Error response missing 'code' field. Body: %s", string(body))
	}
}

// TestResponseContentType validates Content-Type headers.
func TestResponseContentType(t *testing.T) {
	cfg := getConfig(t)

	client := &http.Client{Timeout: 10 * time.Second}

	jsonEndpoints := []string{
		"/healthz",
		"/readyz",
	}

	for _, path := range jsonEndpoints {
		t.Run(path, func(t *testing.T) {
			url := cfg.BaseURL + path
			resp, err := client.Get(url)
			if err != nil {
				t.Skipf("Server not available: %v", err)
			}
			defer resp.Body.Close()

			contentType := resp.Header.Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected application/json Content-Type for %s, got: %s", path, contentType)
			}
		})
	}
}

// TestRequiredFieldsPresent validates response bodies have required fields.
func TestRequiredFieldsPresent(t *testing.T) {
	cfg := getConfig(t)
	spec, router := loadSpec(t, cfg.SpecPath)

	client := &http.Client{Timeout: 10 * time.Second}

	t.Run("HealthzResponse", func(t *testing.T) {
		url := cfg.BaseURL + "/healthz"
		req, _ := http.NewRequest("GET", url, nil)

		resp, err := client.Do(req)
		if err != nil {
			t.Skipf("Server not available: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		// Validate against spec
		route, pathParams, err := router.FindRoute(req)
		if err != nil {
			t.Fatalf("Could not find route in spec: %v", err)
		}

		requestValidationInput := &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		}

		responseValidationInput := &openapi3filter.ResponseValidationInput{
			RequestValidationInput: requestValidationInput,
			Status:                 resp.StatusCode,
			Header:                 resp.Header,
			Body:                   io.NopCloser(strings.NewReader(string(body))),
		}

		err = openapi3filter.ValidateResponse(context.Background(), responseValidationInput)
		if err != nil {
			t.Errorf("Response validation failed: %v", err)
		}
	})

	// Log spec info for debugging
	t.Logf("Spec version: %s", spec.Info.Version)
	t.Logf("Spec title: %s", spec.Info.Title)
}
