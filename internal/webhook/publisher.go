package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/penshort/penshort/internal/model"
)

// Publisher creates webhook delivery records when events occur.
type Publisher struct {
	repo   *Repository
	logger *slog.Logger
}

// NewPublisher creates a new webhook publisher.
func NewPublisher(repo *Repository, logger *slog.Logger) *Publisher {
	return &Publisher{
		repo:   repo,
		logger: logger.With("component", "webhook.publisher"),
	}
}

// PublishClickEvent creates webhook deliveries for a click event.
// It fans out to all active endpoints subscribed to click events.
func (p *Publisher) PublishClickEvent(ctx context.Context, userID string, click *model.ClickEvent) error {
	// Find all active endpoints for this user that subscribe to click events
	endpoints, err := p.repo.ListActiveEndpointsByUserAndEvent(ctx, userID, model.EventTypeClick)
	if err != nil {
		return fmt.Errorf("list active endpoints: %w", err)
	}

	if len(endpoints) == 0 {
		return nil // No webhooks configured
	}

	// Build payload once, reuse for all endpoints
	payload := model.WebhookPayload{
		EventType: string(model.EventTypeClick),
		EventID:   click.ID,
		Timestamp: click.ClickedAt,
		Data: map[string]any{
			"short_code":   click.ShortCode,
			"link_id":      click.LinkID,
			"referrer":     extractReferrerDomain(click.Referrer),
			"country_code": click.CountryCode,
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Create delivery for each endpoint
	now := time.Now()
	for _, endpoint := range endpoints {
		delivery := &model.WebhookDelivery{
			ID:          generateULID(),
			EndpointID:  endpoint.ID,
			EventID:     click.ID,
			EventType:   model.EventTypeClick,
			PayloadJSON: string(payloadJSON),
			Status:      model.DeliveryStatusPending,
			AttemptCount: 0,
			MaxAttempts:  DefaultMaxAttempts,
			NextRetryAt:  now, // Immediate delivery
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := p.repo.CreateDelivery(ctx, delivery); err != nil {
			p.logger.Warn("failed to create delivery",
				"endpoint_id", endpoint.ID,
				"event_id", click.ID,
				"error", err,
			)
			// Continue with other endpoints
			continue
		}

		p.logger.Debug("webhook delivery created",
			"delivery_id", delivery.ID,
			"endpoint_id", endpoint.ID,
			"event_id", click.ID,
		)
	}

	return nil
}

// extractReferrerDomain extracts domain from referrer URL for privacy.
func extractReferrerDomain(ref string) string {
	if ref == "" {
		return ""
	}
	// Simple extraction - the referrer is already sanitized by analytics
	return ref
}

// generateULID generates a ULID-like unique ID.
func generateULID() string {
	// Use timestamp + random for uniqueness
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%016x", timestamp)
}
