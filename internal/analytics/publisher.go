// Package analytics provides click event capture and processing.
package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/penshort/penshort/internal/metrics"
)

const (
	// StreamKey is the Redis stream for click events.
	StreamKey = "stream:click_events"

	// DeadLetterStreamKey is the Redis stream for poison messages.
	DeadLetterStreamKey = "stream:click_events:dlq"

	// MaxStreamLen is the approximate max length of the stream.
	MaxStreamLen = 100000

	// PublishTimeout is the max time to wait for Redis publish.
	PublishTimeout = 100 * time.Millisecond
)

// ClickEventPayload is the compressed event format for Redis stream.
type ClickEventPayload struct {
	ShortCode   string `json:"sc"`           // short_code
	LinkID      string `json:"lid"`          // link_id
	Referrer    string `json:"r,omitempty"`  // referrer (truncated)
	UserAgent   string `json:"ua,omitempty"` // user_agent (truncated)
	VisitorHash string `json:"vh"`           // visitor_hash
	CountryCode string `json:"cc,omitempty"` // country_code
	ClickedAt   int64  `json:"t"`            // Unix milliseconds
}

// Publisher enqueues click events to Redis stream.
type Publisher struct {
	redis   *redis.Client
	logger  *slog.Logger
	metrics metrics.Recorder
}

// NewPublisher creates a new analytics event publisher.
func NewPublisher(client *redis.Client, logger *slog.Logger, recorder metrics.Recorder) *Publisher {
	if recorder == nil {
		recorder = metrics.NewNoop()
	}
	return &Publisher{
		redis:   client,
		logger:  logger.With("component", "analytics.publisher"),
		metrics: recorder,
	}
}

// Publish adds a click event to the stream synchronously.
func (p *Publisher) Publish(ctx context.Context, event ClickEventPayload) (string, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return "", fmt.Errorf("marshal event: %w", err)
	}

	result, err := p.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		MaxLen: MaxStreamLen,
		Approx: true, // ~MAXLEN for performance
		ID:     "*",  // Auto-generate ID
		Values: map[string]interface{}{
			"payload": string(data),
		},
	}).Result()

	if err != nil {
		return "", fmt.Errorf("xadd: %w", err)
	}

	return result, nil
}

// PublishAsync publishes without blocking the caller.
// Errors are logged but not returned (fire-and-forget).
func (p *Publisher) PublishAsync(event ClickEventPayload) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), PublishTimeout)
		defer cancel()

		streamID, err := p.Publish(ctx, event)
		if err != nil {
			p.logger.Warn("failed to publish click event",
				"short_code", event.ShortCode,
				"error", err,
			)
			p.metrics.IncAnalyticsEventPublished("dropped")
			return
		}

		p.logger.Debug("click event published",
			"short_code", event.ShortCode,
			"stream_id", streamID,
		)
		p.metrics.IncAnalyticsEventPublished("success")
	}()
}

// GenerateVisitorHash creates a privacy-safe visitor identifier.
// Uses SHA256(IP + UserAgent + daily_salt) truncated to 16 hex chars.
func GenerateVisitorHash(ip, userAgent string, clickedAt time.Time) string {
	// Daily salt rotates at midnight UTC
	dailySalt := fmt.Sprintf("penshort:%s", clickedAt.UTC().Format("2006-01-02"))

	data := ip + userAgent + dailySalt
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:16]
}

// SanitizeReferrer cleans and truncates the referrer URL.
// Strips query parameters and fragments for privacy.
func SanitizeReferrer(ref string) string {
	if ref == "" {
		return ""
	}

	parsed, err := url.Parse(ref)
	if err != nil {
		return ""
	}

	// Keep only scheme + host + path; strip query params and fragments
	parsed.RawQuery = ""
	parsed.Fragment = ""

	sanitized := parsed.String()
	if len(sanitized) > 500 {
		return sanitized[:500]
	}
	return sanitized
}

// TruncateUserAgent truncates user agent to max 500 chars.
func TruncateUserAgent(ua string) string {
	if len(ua) > 500 {
		return ua[:500]
	}
	return ua
}

// ExtractCountryCode extracts country code from Cloudflare header.
// Returns empty string if header is missing or invalid.
func ExtractCountryCode(cfIPCountry string) string {
	if cfIPCountry != "" && len(cfIPCountry) == 2 {
		return strings.ToUpper(cfIPCountry)
	}
	return ""
}

// ExtractReferrerDomain extracts the domain from a referrer URL.
// Returns "(direct)" for empty referrer.
func ExtractReferrerDomain(ref string) string {
	if ref == "" {
		return "(direct)"
	}

	parsed, err := url.Parse(ref)
	if err != nil || parsed.Host == "" {
		return "(unknown)"
	}

	return parsed.Host
}
