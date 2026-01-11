// Package model defines domain entities for the application.
package model

import (
	"slices"
	"time"
)

// EventType represents webhook event types.
type EventType string

const (
	EventTypeClick EventType = "click"
	// Future: EventTypeLinkCreated, EventTypeLinkExpired, etc.
)

// ValidEventTypes contains all valid event types.
var ValidEventTypes = []EventType{EventTypeClick}

// IsValidEventType checks if an event type is valid.
func IsValidEventType(et EventType) bool {
	return slices.Contains(ValidEventTypes, et)
}

// DeliveryStatus represents webhook delivery state.
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusSuccess   DeliveryStatus = "success"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusExhausted DeliveryStatus = "exhausted"
)

// WebhookEndpoint represents a webhook configuration.
type WebhookEndpoint struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	TargetURL   string      `json:"target_url"`
	SecretHash  string      `json:"-"` // Never expose
	Enabled     bool        `json:"enabled"`
	EventTypes  []EventType `json:"event_types"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	DeletedAt   *time.Time  `json:"-"`
}

// IsDeleted returns true if the endpoint is soft-deleted.
func (e *WebhookEndpoint) IsDeleted() bool {
	return e.DeletedAt != nil
}

// IsActive returns true if the endpoint can receive webhooks.
func (e *WebhookEndpoint) IsActive() bool {
	return e.Enabled && !e.IsDeleted()
}

// SubscribesToEvent checks if endpoint subscribes to given event type.
func (e *WebhookEndpoint) SubscribesToEvent(et EventType) bool {
	return slices.Contains(e.EventTypes, et)
}

// WebhookDelivery represents a delivery attempt record.
type WebhookDelivery struct {
	ID             string         `json:"id"`
	EndpointID     string         `json:"endpoint_id"`
	EventID        string         `json:"event_id"`
	EventType      EventType      `json:"event_type"`
	PayloadJSON    string         `json:"-"` // Don't expose full payload in API
	Status         DeliveryStatus `json:"status"`
	AttemptCount   int            `json:"attempt_count"`
	MaxAttempts    int            `json:"max_attempts"`
	NextRetryAt    time.Time      `json:"next_retry_at,omitempty"`
	LastAttemptAt  *time.Time     `json:"last_attempt_at,omitempty"`
	LastHTTPStatus *int           `json:"last_http_status,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// CanRetry returns true if delivery can be retried.
func (d *WebhookDelivery) CanRetry() bool {
	return d.Status == DeliveryStatusFailed && d.AttemptCount < d.MaxAttempts
}

// IsTerminal returns true if delivery is in a terminal state.
func (d *WebhookDelivery) IsTerminal() bool {
	return d.Status == DeliveryStatusSuccess || d.Status == DeliveryStatusExhausted
}

// WebhookEndpointCreateRequest represents request to create webhook endpoint.
type WebhookEndpointCreateRequest struct {
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	TargetURL   string      `json:"target_url"`
	EventTypes  []EventType `json:"event_types,omitempty"` // Defaults to ["click"]
}

// WebhookEndpointUpdateRequest represents request to update webhook endpoint.
type WebhookEndpointUpdateRequest struct {
	Name        *string      `json:"name,omitempty"`
	Description *string      `json:"description,omitempty"`
	TargetURL   *string      `json:"target_url,omitempty"`
	Enabled     *bool        `json:"enabled,omitempty"`
	EventTypes  *[]EventType `json:"event_types,omitempty"`
}

// WebhookEndpointResponse represents API response for webhook endpoint.
type WebhookEndpointResponse struct {
	ID          string      `json:"id"`
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	TargetURL   string      `json:"target_url"`
	Enabled     bool        `json:"enabled"`
	EventTypes  []EventType `json:"event_types"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ToResponse converts WebhookEndpoint to API response.
func (e *WebhookEndpoint) ToResponse() WebhookEndpointResponse {
	return WebhookEndpointResponse{
		ID:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		TargetURL:   e.TargetURL,
		Enabled:     e.Enabled,
		EventTypes:  e.EventTypes,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// WebhookEndpointCreateResponse includes secret (shown only once).
type WebhookEndpointCreateResponse struct {
	WebhookEndpointResponse
	Secret string `json:"secret"` // Plaintext - display once only!
}

// WebhookDeliveryResponse represents API response for delivery.
type WebhookDeliveryResponse struct {
	ID             string         `json:"id"`
	EventID        string         `json:"event_id"`
	EventType      EventType      `json:"event_type"`
	Status         DeliveryStatus `json:"status"`
	AttemptCount   int            `json:"attempt_count"`
	MaxAttempts    int            `json:"max_attempts"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	LastAttemptAt  *time.Time     `json:"last_attempt_at,omitempty"`
	LastHTTPStatus *int           `json:"last_http_status,omitempty"`
	LastError      string         `json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ToResponse converts WebhookDelivery to API response.
func (d *WebhookDelivery) ToResponse() WebhookDeliveryResponse {
	resp := WebhookDeliveryResponse{
		ID:             d.ID,
		EventID:        d.EventID,
		EventType:      d.EventType,
		Status:         d.Status,
		AttemptCount:   d.AttemptCount,
		MaxAttempts:    d.MaxAttempts,
		LastAttemptAt:  d.LastAttemptAt,
		LastHTTPStatus: d.LastHTTPStatus,
		LastError:      d.LastError,
		CreatedAt:      d.CreatedAt,
	}
	if !d.NextRetryAt.IsZero() && !d.IsTerminal() {
		resp.NextRetryAt = &d.NextRetryAt
	}
	return resp
}

// WebhookPayload represents the payload sent to webhook endpoints.
type WebhookPayload struct {
	EventType string         `json:"event_type"`
	EventID   string         `json:"event_id"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// ClickEventData represents the data field for click events.
type ClickEventData struct {
	ShortCode   string `json:"short_code"`
	LinkID      string `json:"link_id"`
	Referrer    string `json:"referrer,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
}
