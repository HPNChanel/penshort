//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
)

const (
	systemUserID = "system"
	systemEmail  = "system@penshort.local"
)

type apiKeyCreateResponse struct {
	ID     string   `json:"id"`
	Key    string   `json:"key"`
	Scopes []string `json:"scopes"`
}

type linkResponse struct {
	ID          string `json:"id"`
	ShortCode   string `json:"short_code"`
	Destination string `json:"destination"`
}

type webhookCreateResponse struct {
	ID        string `json:"id"`
	TargetURL string `json:"target_url"`
	Secret    string `json:"secret"`
}

type webhookRequest struct {
	Headers http.Header
	Body    []byte
}

func TestE2ESmoke(t *testing.T) {
	baseURL := envOrDefault("PENSHORT_BASE_URL", "http://localhost:8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL is required for e2e tests")
	}

	bootstrapKey := bootstrapAdminKey(t, dbURL)
	testKey := createAPIKey(t, baseURL, bootstrapKey)

	link := createLink(t, baseURL, testKey)

	assertRedirect(t, baseURL, link.ShortCode, link.Destination)
	waitForAnalytics(t, baseURL, testKey, link.ID)

	webhookURL, deliveries, shutdown := startWebhookReceiver(t)
	defer shutdown()
	createWebhookEndpoint(t, baseURL, testKey, webhookURL)

	assertRedirect(t, baseURL, link.ShortCode, link.Destination)
	waitForWebhookDelivery(t, deliveries, link.ShortCode)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func bootstrapAdminKey(t *testing.T, dbURL string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := repository.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer repo.Close()

	if err := ensureUser(ctx, repo, systemUserID, systemEmail); err != nil {
		t.Fatalf("ensure user: %v", err)
	}

	generated, err := auth.GenerateAPIKey(auth.EnvLive)
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}

	apiKey := &model.APIKey{
		ID:            ulid.Make().String(),
		UserID:        systemUserID,
		KeyHash:       generated.Hash,
		KeyPrefix:     generated.Prefix,
		Scopes:        []string{model.ScopeAdmin},
		RateLimitTier: model.TierUnlimited,
		Name:          "e2e-bootstrap",
		CreatedAt:     time.Now().UTC(),
	}

	if err := repo.CreateAPIKey(ctx, apiKey); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	return generated.Plaintext
}

func ensureUser(ctx context.Context, repo *repository.Repository, userID, email string) error {
	if existing, err := repo.GetUserByID(ctx, userID); err == nil {
		if existing.Email != email {
			return fmt.Errorf("user %s exists with different email: %s", userID, existing.Email)
		}
		return nil
	}

	if byEmail, err := repo.GetUserByEmail(ctx, email); err == nil {
		if byEmail.ID != userID {
			return fmt.Errorf("email %s already used by user %s", email, byEmail.ID)
		}
		return nil
	}

	user := &model.User{ID: userID, Email: email, CreatedAt: time.Now().UTC()}
	return repo.CreateUser(ctx, user)
}

func createAPIKey(t *testing.T, baseURL, bootstrapKey string) string {
	t.Helper()

	payload := map[string]any{
		"name":   "e2e-key",
		"scopes": []string{"admin"},
	}

	var resp apiKeyCreateResponse
	status := doJSON(t, http.MethodPost, baseURL+"/api/v1/api-keys", bootstrapKey, payload, &resp)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from api key create, got %d", status)
	}
	if resp.Key == "" {
		t.Fatalf("api key response missing key")
	}
	return resp.Key
}

func createLink(t *testing.T, baseURL, apiKey string) linkResponse {
	t.Helper()

	alias := fmt.Sprintf("e2e-%d", time.Now().UnixNano())
	payload := map[string]any{
		"destination": "https://example.com/e2e",
		"alias":       alias,
		"redirect_type": 302,
	}

	var resp linkResponse
	status := doJSON(t, http.MethodPost, baseURL+"/api/v1/links", apiKey, payload, &resp)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from link create, got %d", status)
	}
	if resp.ID == "" || resp.ShortCode == "" {
		t.Fatalf("link create response missing fields")
	}
	return resp
}

func assertRedirect(t *testing.T, baseURL, shortCode, destination string) {
	t.Helper()

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", baseURL, shortCode), nil)
	if err != nil {
		t.Fatalf("create redirect request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("redirect request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("expected redirect status, got %d", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location != destination {
		t.Fatalf("expected Location %q, got %q", destination, location)
	}
}

func waitForAnalytics(t *testing.T, baseURL, apiKey, linkID string) {
	t.Helper()

	from := time.Now().UTC().Format("2006-01-02")
	to := from
	endpoint := fmt.Sprintf("%s/api/v1/links/%s/analytics?from=%s&to=%s", baseURL, linkID, from, to)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var resp model.AnalyticsResponse
		status := doJSON(t, http.MethodGet, endpoint, apiKey, nil, &resp)
		if status == http.StatusOK && resp.Summary.TotalClicks >= 1 {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("analytics did not report clicks in time")
}

func startWebhookReceiver(t *testing.T) (string, <-chan webhookRequest, func()) {
	t.Helper()

	received := make(chan webhookRequest, 1)

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen webhook: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		received <- webhookRequest{Headers: r.Header.Clone(), Body: body}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Handler: handler}
	go func() {
		_ = srv.Serve(listener)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://host.docker.internal:%d/webhook", port)

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}

	return url, received, shutdown
}

func createWebhookEndpoint(t *testing.T, baseURL, apiKey, targetURL string) {
	t.Helper()

	payload := map[string]any{
		"target_url": targetURL,
		"event_types": []string{"click"},
		"name": "e2e-webhook",
	}

	var resp webhookCreateResponse
	status := doJSON(t, http.MethodPost, baseURL+"/api/v1/webhooks", apiKey, payload, &resp)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from webhook create, got %d", status)
	}
	if resp.ID == "" || resp.Secret == "" {
		t.Fatalf("webhook create response missing fields")
	}
}

func waitForWebhookDelivery(t *testing.T, deliveries <-chan webhookRequest, shortCode string) {
	t.Helper()

	select {
	case req := <-deliveries:
		if req.Headers.Get("X-Penshort-Signature") == "" {
			t.Fatalf("missing X-Penshort-Signature header")
		}
		if req.Headers.Get("X-Penshort-Timestamp") == "" {
			t.Fatalf("missing X-Penshort-Timestamp header")
		}
		if req.Headers.Get("X-Penshort-Delivery-Id") == "" {
			t.Fatalf("missing X-Penshort-Delivery-Id header")
		}

		var payload model.WebhookPayload
		if err := json.Unmarshal(req.Body, &payload); err != nil {
			t.Fatalf("decode webhook payload: %v", err)
		}
		if payload.EventType != string(model.EventTypeClick) {
			t.Fatalf("unexpected event_type %q", payload.EventType)
		}
		if payload.Data == nil {
			t.Fatalf("webhook payload missing data")
		}
		if sc, ok := payload.Data["short_code"].(string); !ok || sc != shortCode {
			t.Fatalf("unexpected short_code in webhook payload")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for webhook delivery")
	}
}

func doJSON(t *testing.T, method, url, apiKey string, body any, out any) int {
	t.Helper()

	var buf io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		buf = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if out != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(out); err != nil && resp.ContentLength != 0 {
			t.Fatalf("decode response: %v", err)
		}
	}

	return resp.StatusCode
}

// TestE2ELinkExpiry validates that expired links return 410 Gone or 404.
func TestE2ELinkExpiry(t *testing.T) {
	baseURL := envOrDefault("PENSHORT_BASE_URL", "http://localhost:8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL is required for e2e tests")
	}

	bootstrapKey := bootstrapAdminKey(t, dbURL)
	testKey := createAPIKey(t, baseURL, bootstrapKey)

	// Create link with 3-second expiry
	expiresAt := time.Now().Add(3 * time.Second)
	alias := fmt.Sprintf("e2e-expiry-%d", time.Now().UnixNano())

	payload := map[string]any{
		"destination":   "https://example.com/expiry-test",
		"alias":         alias,
		"redirect_type": 302,
		"expires_at":    expiresAt.Format(time.RFC3339),
	}

	var link linkResponse
	status := doJSON(t, http.MethodPost, baseURL+"/api/v1/links", testKey, payload, &link)
	if status != http.StatusCreated {
		t.Fatalf("expected 201 from link create, got %d", status)
	}

	// Verify redirect works BEFORE expiry
	assertRedirect(t, baseURL, link.ShortCode, "https://example.com/expiry-test")

	// Wait for expiry
	time.Sleep(4 * time.Second)

	// Verify link is expired (should return 410 Gone or 404 Not Found)
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", baseURL, link.ShortCode), nil)
	if err != nil {
		t.Fatalf("create expired link request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expired link request: %v", err)
	}
	defer resp.Body.Close()

	// Expired links should return 410 Gone or 404 Not Found
	if resp.StatusCode != http.StatusGone && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 410 or 404 for expired link, got %d", resp.StatusCode)
	}
}

// TestE2ERateLimiting validates that rate limiting returns 429 with proper headers.
func TestE2ERateLimiting(t *testing.T) {
	baseURL := envOrDefault("PENSHORT_BASE_URL", "http://localhost:8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL is required for e2e tests")
	}

	// Create a free-tier API key (60 RPM, 10 burst)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repo, err := repository.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer repo.Close()

	if err := ensureUser(ctx, repo, systemUserID, systemEmail); err != nil {
		t.Fatalf("ensure user: %v", err)
	}

	generated, err := auth.GenerateAPIKey(auth.EnvLive)
	if err != nil {
		t.Fatalf("generate api key: %v", err)
	}

	apiKey := &model.APIKey{
		ID:            ulid.Make().String(),
		UserID:        systemUserID,
		KeyHash:       generated.Hash,
		KeyPrefix:     generated.Prefix,
		Scopes:        []string{model.ScopeRead},
		RateLimitTier: model.TierFree, // Free tier: 60 RPM, burst 10
		Name:          "e2e-ratelimit-test",
		CreatedAt:     time.Now().UTC(),
	}

	if err := repo.CreateAPIKey(ctx, apiKey); err != nil {
		t.Fatalf("create free-tier api key: %v", err)
	}

	testKey := generated.Plaintext

	// Send requests until we hit rate limit
	client := &http.Client{Timeout: 10 * time.Second}
	var rateLimited bool
	var lastResp *http.Response

	// Free tier has burst of 10, try 20 requests rapidly
	for i := 0; i < 20; i++ {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/links", nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+testKey)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimited = true
			lastResp = resp
			break
		}
		resp.Body.Close()
	}

	if !rateLimited {
		t.Fatalf("expected 429 rate limit after burst, but never hit rate limit")
	}

	defer lastResp.Body.Close()

	// Verify rate limit headers
	limitHeader := lastResp.Header.Get("X-RateLimit-Limit")
	remainingHeader := lastResp.Header.Get("X-RateLimit-Remaining")
	retryAfterHeader := lastResp.Header.Get("Retry-After")

	if limitHeader == "" {
		t.Error("missing X-RateLimit-Limit header on 429 response")
	}
	if remainingHeader != "0" {
		t.Errorf("expected X-RateLimit-Remaining=0, got %s", remainingHeader)
	}
	if retryAfterHeader == "" {
		t.Log("Retry-After header not present (optional but recommended)")
	}

	// Verify response body
	var errResp map[string]any
	if err := json.NewDecoder(lastResp.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode 429 response: %v", err)
	}

	if errResp["error"] == nil {
		t.Error("429 response missing 'error' field")
	}
}

// TestE2ENoSecretsInLogs validates that API keys are not leaked in responses.
// This test validates that error responses don't echo back sensitive credentials.
func TestE2ENoSecretsInLogs(t *testing.T) {
	baseURL := envOrDefault("PENSHORT_BASE_URL", "http://localhost:8080")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatalf("DATABASE_URL is required for e2e tests")
	}

	bootstrapKey := bootstrapAdminKey(t, dbURL)

	client := &http.Client{Timeout: 10 * time.Second}

	// Test that error responses don't leak the Authorization header value
	testKey := "pk_live_fake_" + strings.Repeat("x", 32)
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/links", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	bodyStr := string(body)

	// The fake API key should NEVER appear in error responses
	if strings.Contains(bodyStr, testKey) {
		t.Error("SECURITY: Error response leaked Authorization header value")
	}

	// The bootstrap key should never be echoed back
	if strings.Contains(bodyStr, bootstrapKey) {
		t.Error("SECURITY: Response contains the bootstrap API key")
	}

	// Test with a valid key - responses should not include the key itself
	req2, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/links", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req2.Header.Set("Authorization", "Bearer "+bootstrapKey)

	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	// The full API key should never appear in successful responses
	if strings.Contains(string(body2), bootstrapKey) {
		t.Error("SECURITY: Successful response echoed back the API key")
	}
}
