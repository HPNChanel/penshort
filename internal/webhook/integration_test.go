package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockWebhookReceiver simulates a webhook endpoint for testing.
type MockWebhookReceiver struct {
	Server       *httptest.Server
	Secret       string
	Deliveries   []ReceivedDelivery
	FailCount    int32         // Number of times to fail before succeeding
	failCounter  int32         // Current fail count
	ResponseCode int           // Default response code (200 if 0)
	mu           sync.Mutex
}

// ReceivedDelivery represents a webhook delivery received by the mock server.
type ReceivedDelivery struct {
	Signature    string
	Timestamp    int64
	DeliveryID   string
	Payload      json.RawMessage
	ReceivedAt   time.Time
	SignatureOK  bool
}

// NewMockWebhookReceiver creates a mock webhook receiver.
func NewMockWebhookReceiver(secret string) *MockWebhookReceiver {
	mr := &MockWebhookReceiver{
		Secret:       secret,
		Deliveries:   make([]ReceivedDelivery, 0),
		ResponseCode: http.StatusOK,
	}

	mr.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mr.handleRequest(w, r)
	}))

	return mr
}

// SetFailCount sets how many times the server should fail before succeeding.
func (mr *MockWebhookReceiver) SetFailCount(count int32) {
	atomic.StoreInt32(&mr.FailCount, count)
	atomic.StoreInt32(&mr.failCounter, 0)
}

func (mr *MockWebhookReceiver) handleRequest(w http.ResponseWriter, r *http.Request) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Check if we should fail
	if atomic.LoadInt32(&mr.failCounter) < atomic.LoadInt32(&mr.FailCount) {
		atomic.AddInt32(&mr.failCounter, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Parse headers
	signature := r.Header.Get(HeaderSignature)
	timestampStr := r.Header.Get(HeaderTimestamp)
	deliveryID := r.Header.Get(HeaderDeliveryID)

	timestamp, _ := strconv.ParseInt(timestampStr, 10, 64)

	// Read payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify signature
	signatureOK := mr.verifySignature(signature, timestamp, body)

	delivery := ReceivedDelivery{
		Signature:   signature,
		Timestamp:   timestamp,
		DeliveryID:  deliveryID,
		Payload:     body,
		ReceivedAt:  time.Now(),
		SignatureOK: signatureOK,
	}

	mr.Deliveries = append(mr.Deliveries, delivery)

	responseCode := mr.ResponseCode
	if responseCode == 0 {
		responseCode = http.StatusOK
	}
	w.WriteHeader(responseCode)
}

// verifySignature verifies the HMAC-SHA256 signature.
func (mr *MockWebhookReceiver) verifySignature(signature string, timestamp int64, payload []byte) bool {
	canonical := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(mr.Secret))
	mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// GetDeliveries returns a copy of received deliveries.
func (mr *MockWebhookReceiver) GetDeliveries() []ReceivedDelivery {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	return append([]ReceivedDelivery{}, mr.Deliveries...)
}

// Close shuts down the mock server.
func (mr *MockWebhookReceiver) Close() {
	mr.Server.Close()
}

// TestMockWebhookReceiver_SignatureVerification tests that the mock receiver
// correctly verifies HMAC-SHA256 signatures.
func TestMockWebhookReceiver_SignatureVerification(t *testing.T) {
	secret := "whsec_test_secret_12345"
	receiver := NewMockWebhookReceiver(secret)
	defer receiver.Close()

	// Create a test payload
	payload := []byte(`{"event_type":"click","event_id":"test123"}`)
	timestamp := time.Now().Unix()

	// Generate signature using our signer
	signature := GenerateSignature(secret, timestamp, payload)

	// Send request to mock server
	req, _ := http.NewRequest(http.MethodPost, receiver.Server.URL, nil)
	req.Body = io.NopCloser(bytesReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderSignature, signature)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(HeaderDeliveryID, "delivery_001")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify the mock received and validated the signature
	deliveries := receiver.GetDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}

	if !deliveries[0].SignatureOK {
		t.Error("signature verification failed at receiver")
	}

	if deliveries[0].DeliveryID != "delivery_001" {
		t.Errorf("delivery ID mismatch: got %q", deliveries[0].DeliveryID)
	}
}

// TestMockWebhookReceiver_InvalidSignature tests that invalid signatures are detected.
func TestMockWebhookReceiver_InvalidSignature(t *testing.T) {
	secret := "whsec_real_secret"
	receiver := NewMockWebhookReceiver(secret)
	defer receiver.Close()

	payload := []byte(`{"event_type":"click"}`)
	timestamp := time.Now().Unix()

	// Generate signature with WRONG secret
	wrongSignature := GenerateSignature("wrong_secret", timestamp, payload)

	req, _ := http.NewRequest(http.MethodPost, receiver.Server.URL, nil)
	req.Body = io.NopCloser(bytesReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderSignature, wrongSignature)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(HeaderDeliveryID, "delivery_002")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	deliveries := receiver.GetDeliveries()
	if len(deliveries) != 1 {
		t.Fatalf("expected 1 delivery, got %d", len(deliveries))
	}

	// The signature should NOT be valid
	if deliveries[0].SignatureOK {
		t.Error("expected signature verification to fail with wrong secret")
	}
}

// TestMockWebhookReceiver_RetryScenario tests the fail-then-succeed pattern.
func TestMockWebhookReceiver_RetryScenario(t *testing.T) {
	secret := "whsec_retry_test"
	receiver := NewMockWebhookReceiver(secret)
	defer receiver.Close()

	// Server fails first 2 times, then succeeds
	receiver.SetFailCount(2)

	payload := []byte(`{"event_type":"click"}`)
	timestamp := time.Now().Unix()
	signature := GenerateSignature(secret, timestamp, payload)

	client := &http.Client{Timeout: 5 * time.Second}

	// First attempt - should fail
	req1, _ := http.NewRequest(http.MethodPost, receiver.Server.URL, nil)
	req1.Body = io.NopCloser(bytesReader(payload))
	req1.Header.Set(HeaderSignature, signature)
	req1.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req1.Header.Set(HeaderDeliveryID, "retry_test_1")

	resp1, _ := client.Do(req1)
	if resp1.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("attempt 1: expected 503, got %d", resp1.StatusCode)
	}
	resp1.Body.Close()

	// Second attempt - should fail
	req2, _ := http.NewRequest(http.MethodPost, receiver.Server.URL, nil)
	req2.Body = io.NopCloser(bytesReader(payload))
	req2.Header.Set(HeaderSignature, signature)
	req2.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req2.Header.Set(HeaderDeliveryID, "retry_test_2")

	resp2, _ := client.Do(req2)
	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("attempt 2: expected 503, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	// Third attempt - should succeed
	req3, _ := http.NewRequest(http.MethodPost, receiver.Server.URL, nil)
	req3.Body = io.NopCloser(bytesReader(payload))
	req3.Header.Set(HeaderSignature, signature)
	req3.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req3.Header.Set(HeaderDeliveryID, "retry_test_3")

	resp3, _ := client.Do(req3)
	if resp3.StatusCode != http.StatusOK {
		t.Errorf("attempt 3: expected 200, got %d", resp3.StatusCode)
	}
	resp3.Body.Close()

	// Only the third delivery should have been recorded (fails don't add to deliveries list until success handler)
	deliveries := receiver.GetDeliveries()
	if len(deliveries) != 1 {
		t.Errorf("expected 1 successful delivery, got %d", len(deliveries))
	}
}

// TestCanonicalStringFormat tests the canonical string format used for signing.
func TestCanonicalStringFormat(t *testing.T) {
	// This test documents and verifies the exact canonical string format
	secret := "test_secret"
	timestamp := int64(1736600000)
	payload := []byte(`{"event_type":"click","data":{"short_code":"abc123"}}`)

	// Expected canonical string: "{timestamp}.{payload}"
	expectedCanonical := `1736600000.{"event_type":"click","data":{"short_code":"abc123"}}`

	// Manually compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(expectedCanonical))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare with our GenerateSignature function
	actualSig := GenerateSignature(secret, timestamp, payload)

	if actualSig != expectedSig {
		t.Errorf("signature mismatch\nexpected: %s\nactual: %s", expectedSig, actualSig)
	}

	// Verify the expected values for documentation
	t.Logf("Canonical string format: {timestamp}.{payload}")
	t.Logf("Example: %s", expectedCanonical)
	t.Logf("Signature: %s", expectedSig)
}

// TestHTTPClient_SecurityConfiguration tests the HTTP client security settings.
func TestHTTPClient_SecurityConfiguration(t *testing.T) {
	client := NewHTTPClient()

	if client.Timeout != 30*time.Second {
		t.Errorf("expected 30s timeout, got %v", client.Timeout)
	}

	// Verify redirect handling
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/redirect", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer redirectServer.Close()

	resp, err := client.Get(redirectServer.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Should NOT follow redirects
	if resp.StatusCode != http.StatusFound {
		t.Errorf("client should not follow redirects, got status %d", resp.StatusCode)
	}
}

// bytesReader is a helper to create an io.Reader from []byte.
func bytesReader(b []byte) *bytesReaderImpl {
	return &bytesReaderImpl{data: b}
}

type bytesReaderImpl struct {
	data []byte
	pos  int
}

func (r *bytesReaderImpl) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
