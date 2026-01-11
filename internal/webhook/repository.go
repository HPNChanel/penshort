package webhook

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/penshort/penshort/internal/model"
)

// Repository handles webhook database operations.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new webhook repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateEndpoint creates a new webhook endpoint.
func (r *Repository) CreateEndpoint(ctx context.Context, endpoint *model.WebhookEndpoint) error {
	query := `
		INSERT INTO webhook_endpoints (
			id, user_id, target_url, secret_hash, enabled, 
			event_types, name, description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	eventTypes := make([]string, len(endpoint.EventTypes))
	for i, et := range endpoint.EventTypes {
		eventTypes[i] = string(et)
	}

	_, err := r.db.ExecContext(ctx, query,
		endpoint.ID,
		endpoint.UserID,
		endpoint.TargetURL,
		endpoint.SecretHash,
		endpoint.Enabled,
		pq.Array(eventTypes),
		endpoint.Name,
		endpoint.Description,
		endpoint.CreatedAt,
		endpoint.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert webhook endpoint: %w", err)
	}
	return nil
}

// GetEndpoint retrieves a webhook endpoint by ID.
func (r *Repository) GetEndpoint(ctx context.Context, id string) (*model.WebhookEndpoint, error) {
	query := `
		SELECT id, user_id, target_url, secret_hash, enabled, event_types,
			   name, description, created_at, updated_at, deleted_at
		FROM webhook_endpoints
		WHERE id = $1 AND deleted_at IS NULL
	`

	var endpoint model.WebhookEndpoint
	var eventTypes []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&endpoint.ID,
		&endpoint.UserID,
		&endpoint.TargetURL,
		&endpoint.SecretHash,
		&endpoint.Enabled,
		pq.Array(&eventTypes),
		&endpoint.Name,
		&endpoint.Description,
		&endpoint.CreatedAt,
		&endpoint.UpdatedAt,
		&endpoint.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrEndpointNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query webhook endpoint: %w", err)
	}

	endpoint.EventTypes = make([]model.EventType, len(eventTypes))
	for i, et := range eventTypes {
		endpoint.EventTypes[i] = model.EventType(et)
	}

	return &endpoint, nil
}

// ListEndpointsByUser retrieves all webhook endpoints for a user.
func (r *Repository) ListEndpointsByUser(ctx context.Context, userID string) ([]*model.WebhookEndpoint, error) {
	query := `
		SELECT id, user_id, target_url, secret_hash, enabled, event_types,
			   name, description, created_at, updated_at
		FROM webhook_endpoints
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query webhooks by user: %w", err)
	}
	defer rows.Close()

	var endpoints []*model.WebhookEndpoint
	for rows.Next() {
		var endpoint model.WebhookEndpoint
		var eventTypes []string

		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.UserID,
			&endpoint.TargetURL,
			&endpoint.SecretHash,
			&endpoint.Enabled,
			pq.Array(&eventTypes),
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook endpoint: %w", err)
		}

		endpoint.EventTypes = make([]model.EventType, len(eventTypes))
		for i, et := range eventTypes {
			endpoint.EventTypes[i] = model.EventType(et)
		}

		endpoints = append(endpoints, &endpoint)
	}

	return endpoints, rows.Err()
}

// ListActiveEndpointsByUserAndEvent retrieves enabled endpoints for user/event.
func (r *Repository) ListActiveEndpointsByUserAndEvent(ctx context.Context, userID string, eventType model.EventType) ([]*model.WebhookEndpoint, error) {
	query := `
		SELECT id, user_id, target_url, secret_hash, enabled, event_types,
			   name, description, created_at, updated_at
		FROM webhook_endpoints
		WHERE user_id = $1 
		  AND deleted_at IS NULL 
		  AND enabled = true
		  AND $2 = ANY(event_types)
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, query, userID, string(eventType))
	if err != nil {
		return nil, fmt.Errorf("query active webhooks: %w", err)
	}
	defer rows.Close()

	var endpoints []*model.WebhookEndpoint
	for rows.Next() {
		var endpoint model.WebhookEndpoint
		var eventTypes []string

		if err := rows.Scan(
			&endpoint.ID,
			&endpoint.UserID,
			&endpoint.TargetURL,
			&endpoint.SecretHash,
			&endpoint.Enabled,
			pq.Array(&eventTypes),
			&endpoint.Name,
			&endpoint.Description,
			&endpoint.CreatedAt,
			&endpoint.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan webhook endpoint: %w", err)
		}

		endpoint.EventTypes = make([]model.EventType, len(eventTypes))
		for i, et := range eventTypes {
			endpoint.EventTypes[i] = model.EventType(et)
		}

		endpoints = append(endpoints, &endpoint)
	}

	return endpoints, rows.Err()
}

// UpdateEndpoint updates a webhook endpoint.
func (r *Repository) UpdateEndpoint(ctx context.Context, endpoint *model.WebhookEndpoint) error {
	query := `
		UPDATE webhook_endpoints
		SET target_url = $2, enabled = $3, event_types = $4,
			name = $5, description = $6, updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`

	eventTypes := make([]string, len(endpoint.EventTypes))
	for i, et := range endpoint.EventTypes {
		eventTypes[i] = string(et)
	}

	result, err := r.db.ExecContext(ctx, query,
		endpoint.ID,
		endpoint.TargetURL,
		endpoint.Enabled,
		pq.Array(eventTypes),
		endpoint.Name,
		endpoint.Description,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("update webhook endpoint: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrEndpointNotFound
	}

	return nil
}

// UpdateEndpointSecret updates the secret hash for an endpoint.
func (r *Repository) UpdateEndpointSecret(ctx context.Context, id, secretHash string) error {
	query := `
		UPDATE webhook_endpoints
		SET secret_hash = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, secretHash, time.Now())
	if err != nil {
		return fmt.Errorf("update endpoint secret: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrEndpointNotFound
	}

	return nil
}

// DeleteEndpoint soft-deletes a webhook endpoint.
func (r *Repository) DeleteEndpoint(ctx context.Context, id string) error {
	query := `
		UPDATE webhook_endpoints
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("delete webhook endpoint: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrEndpointNotFound
	}

	return nil
}

// CreateDelivery creates a new delivery record.
func (r *Repository) CreateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (
			id, endpoint_id, event_id, event_type, payload_json,
			status, attempt_count, max_attempts, next_retry_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (event_id, endpoint_id) DO NOTHING
	`

	_, err := r.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.EndpointID,
		delivery.EventID,
		string(delivery.EventType),
		delivery.PayloadJSON,
		string(delivery.Status),
		delivery.AttemptCount,
		delivery.MaxAttempts,
		delivery.NextRetryAt,
		delivery.CreatedAt,
		delivery.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert webhook delivery: %w", err)
	}
	return nil
}

// GetPendingDeliveries retrieves deliveries ready to be sent.
func (r *Repository) GetPendingDeliveries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error) {
	query := `
		SELECT d.id, d.endpoint_id, d.event_id, d.event_type, d.payload_json,
			   d.status, d.attempt_count, d.max_attempts, d.next_retry_at,
			   d.last_attempt_at, d.last_http_status, d.last_error,
			   d.created_at, d.updated_at
		FROM webhook_deliveries d
		JOIN webhook_endpoints e ON d.endpoint_id = e.id
		WHERE d.status IN ('pending', 'failed')
		  AND d.next_retry_at <= $1
		  AND e.deleted_at IS NULL
		  AND e.enabled = true
		ORDER BY d.next_retry_at
		LIMIT $2
		FOR UPDATE OF d SKIP LOCKED
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("query pending deliveries: %w", err)
	}
	defer rows.Close()

	return scanDeliveries(rows)
}

// UpdateDeliverySuccess marks a delivery as successful.
func (r *Repository) UpdateDeliverySuccess(ctx context.Context, id string, httpStatus int) error {
	query := `
		UPDATE webhook_deliveries
		SET status = 'success',
			attempt_count = attempt_count + 1,
			last_attempt_at = $2,
			last_http_status = $3,
			last_error = NULL,
			updated_at = $2
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, now, httpStatus)
	if err != nil {
		return fmt.Errorf("update delivery success: %w", err)
	}
	return nil
}

// UpdateDeliveryFailure marks a delivery as failed and schedules retry.
func (r *Repository) UpdateDeliveryFailure(ctx context.Context, id string, httpStatus *int, errMsg string, nextRetryAt time.Time, exhausted bool) error {
	status := "failed"
	if exhausted {
		status = "exhausted"
	}

	// Truncate error message
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}

	query := `
		UPDATE webhook_deliveries
		SET status = $2,
			attempt_count = attempt_count + 1,
			last_attempt_at = $3,
			last_http_status = $4,
			last_error = $5,
			next_retry_at = $6,
			updated_at = $3
		WHERE id = $1
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, id, status, now, httpStatus, errMsg, nextRetryAt)
	if err != nil {
		return fmt.Errorf("update delivery failure: %w", err)
	}
	return nil
}

// GetDeliveryWithEndpoint retrieves a delivery with its endpoint for sending.
func (r *Repository) GetDeliveryWithEndpoint(ctx context.Context, deliveryID string) (*model.WebhookDelivery, *model.WebhookEndpoint, error) {
	deliveryQuery := `
		SELECT id, endpoint_id, event_id, event_type, payload_json,
			   status, attempt_count, max_attempts, next_retry_at,
			   last_attempt_at, last_http_status, last_error,
			   created_at, updated_at
		FROM webhook_deliveries
		WHERE id = $1
	`

	var delivery model.WebhookDelivery
	var eventType string

	err := r.db.QueryRowContext(ctx, deliveryQuery, deliveryID).Scan(
		&delivery.ID,
		&delivery.EndpointID,
		&delivery.EventID,
		&eventType,
		&delivery.PayloadJSON,
		&delivery.Status,
		&delivery.AttemptCount,
		&delivery.MaxAttempts,
		&delivery.NextRetryAt,
		&delivery.LastAttemptAt,
		&delivery.LastHTTPStatus,
		&delivery.LastError,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("query delivery: %w", err)
	}
	delivery.EventType = model.EventType(eventType)

	endpoint, err := r.GetEndpoint(ctx, delivery.EndpointID)
	if err != nil {
		return nil, nil, err
	}

	return &delivery, endpoint, nil
}

// ListDeliveriesByEndpoint retrieves deliveries for an endpoint with pagination.
func (r *Repository) ListDeliveriesByEndpoint(ctx context.Context, endpointID string, statuses []string, limit, offset int) ([]*model.WebhookDelivery, int, error) {
	var whereClause strings.Builder
	args := []interface{}{endpointID}
	argIdx := 2

	whereClause.WriteString("WHERE endpoint_id = $1")

	if len(statuses) > 0 {
		placeholders := make([]string, len(statuses))
		for i, s := range statuses {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, s)
			argIdx++
		}
		whereClause.WriteString(fmt.Sprintf(" AND status IN (%s)", strings.Join(placeholders, ",")))
	}

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM webhook_deliveries %s`, whereClause.String())
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count deliveries: %w", err)
	}

	// Get deliveries
	query := fmt.Sprintf(`
		SELECT id, endpoint_id, event_id, event_type, payload_json,
			   status, attempt_count, max_attempts, next_retry_at,
			   last_attempt_at, last_http_status, last_error,
			   created_at, updated_at
		FROM webhook_deliveries
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause.String(), argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query deliveries: %w", err)
	}
	defer rows.Close()

	deliveries, err := scanDeliveries(rows)
	if err != nil {
		return nil, 0, err
	}

	return deliveries, total, nil
}

// ResetDeliveryForRetry resets a delivery for manual retry.
func (r *Repository) ResetDeliveryForRetry(ctx context.Context, id string) error {
	query := `
		UPDATE webhook_deliveries
		SET status = 'pending',
			next_retry_at = $2,
			updated_at = $2
		WHERE id = $1 AND status = 'exhausted'
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("reset delivery: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDeliveryNotFound
	}

	return nil
}

// GetQueueDepth returns the count of pending and failed deliveries.
func (r *Repository) GetQueueDepth(ctx context.Context) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM webhook_deliveries
		WHERE status IN ('pending', 'failed')
	`

	var count int64
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count queue depth: %w", err)
	}
	return count, nil
}

func scanDeliveries(rows *sql.Rows) ([]*model.WebhookDelivery, error) {
	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		var eventType, status string

		if err := rows.Scan(
			&d.ID,
			&d.EndpointID,
			&d.EventID,
			&eventType,
			&d.PayloadJSON,
			&status,
			&d.AttemptCount,
			&d.MaxAttempts,
			&d.NextRetryAt,
			&d.LastAttemptAt,
			&d.LastHTTPStatus,
			&d.LastError,
			&d.CreatedAt,
			&d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}

		d.EventType = model.EventType(eventType)
		d.Status = model.DeliveryStatus(status)
		deliveries = append(deliveries, &d)
	}

	return deliveries, rows.Err()
}
