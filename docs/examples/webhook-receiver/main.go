// Penshort Webhook Receiver Example
//
// This is a minimal example of how to receive and verify Penshort webhooks.
//
// Usage:
//   export PENSHORT_WEBHOOK_SECRET="whsec_your_secret_here"
//   go run main.go
//
// Then configure your Penshort webhook to point to http://your-server:9000/webhook

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ClickEvent represents the webhook payload for click events
type ClickEvent struct {
	Event     string  `json:"event"`
	LinkID    string  `json:"link_id"`
	ShortCode string  `json:"short_code"`
	Timestamp string  `json:"timestamp"`
	Visitor   Visitor `json:"visitor"`
}

type Visitor struct {
	Referrer    string `json:"referrer"`
	UserAgent   string `json:"user_agent"`
	CountryCode string `json:"country_code"`
}

func main() {
	secret := os.Getenv("PENSHORT_WEBHOOK_SECRET")
	if secret == "" {
		log.Fatal("PENSHORT_WEBHOOK_SECRET environment variable is required")
	}

	http.HandleFunc("/webhook", webhookHandler(secret))
	http.HandleFunc("/health", healthHandler)

	log.Println("Starting webhook receiver on :9000")
	log.Println("Endpoint: http://localhost:9000/webhook")
	log.Fatal(http.ListenAndServe(":9000", nil))
}

func webhookHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Get signature header
		signature := r.Header.Get("X-Penshort-Signature")
		if signature == "" {
			log.Println("Missing X-Penshort-Signature header")
			http.Error(w, "Missing signature", http.StatusUnauthorized)
			return
		}

		// Verify signature
		if !verifySignature(signature, string(body), secret) {
			log.Println("Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		// Parse event
		var event ClickEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Process the event
		log.Printf("✓ Received %s event for %s", event.Event, event.ShortCode)
		log.Printf("  Link ID:  %s", event.LinkID)
		log.Printf("  Time:     %s", event.Timestamp)
		log.Printf("  Referrer: %s", event.Visitor.Referrer)
		log.Printf("  Country:  %s", event.Visitor.CountryCode)

		// Respond with 200 OK
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	}
}

// verifySignature verifies the HMAC-SHA256 signature from Penshort
//
// Header format: t=1705142400,v1=abc123def456...
// Signed payload: {timestamp}.{body}
func verifySignature(header, body, secret string) bool {
	parts := strings.Split(header, ",")
	if len(parts) != 2 {
		return false
	}

	// Extract timestamp and signature
	var timestamp, signature string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			signature = strings.TrimPrefix(part, "v1=")
		}
	}

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp (±5 min tolerance)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		log.Println("Signature timestamp too old or in future")
		return false
	}

	// Compute expected signature
	signedPayload := timestamp + "." + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	return hmac.Equal([]byte(signature), []byte(expected))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
