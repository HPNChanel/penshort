// Penshort Webhook Receiver Example
//
// Usage:
//   export PENSHORT_WEBHOOK_SECRET="your_secret_here"
//   go run main.go
//
// Then configure your webhook to point to http://host.docker.internal:9000/webhook

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
	"time"
)

type WebhookPayload struct {
	EventType string    `json:"event_type"`
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      struct {
		ShortCode   string `json:"short_code"`
		LinkID      string `json:"link_id"`
		Referrer    string `json:"referrer"`
		CountryCode string `json:"country_code"`
	} `json:"data"`
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

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		signature := r.Header.Get("X-Penshort-Signature")
		timestamp := r.Header.Get("X-Penshort-Timestamp")
		deliveryID := r.Header.Get("X-Penshort-Delivery-Id")
		if signature == "" || timestamp == "" {
			log.Println("Missing signature headers")
			http.Error(w, "Missing signature", http.StatusUnauthorized)
			return
		}

		if !verifySignature(signature, timestamp, body, secret) {
			log.Println("Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		var payload WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Printf("Received %s event for %s", payload.EventType, payload.Data.ShortCode)
		log.Printf("  Delivery ID: %s", deliveryID)
		log.Printf("  Link ID:     %s", payload.Data.LinkID)
		log.Printf("  Timestamp:   %s", payload.Timestamp.Format(time.RFC3339))
		log.Printf("  Referrer:    %s", payload.Data.Referrer)
		log.Printf("  Country:     %s", payload.Data.CountryCode)

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "received"})
	}
}

func verifySignature(signature, timestamp string, body []byte, secret string) bool {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if math.Abs(float64(time.Now().Unix()-ts)) > 300 {
		return false
	}

	hash := sha256.Sum256([]byte(secret))
	canonical := fmt.Sprintf("%d.%s", ts, string(body))
	mac := hmac.New(sha256.New, hash[:])
	mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
