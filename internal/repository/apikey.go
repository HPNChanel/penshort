package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/penshort/penshort/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

// Common errors for API key repository operations.
var (
	ErrAPIKeyNotFound = errors.New("API key not found")
)

// CreateAPIKey inserts a new API key into the database.
func (r *Repository) CreateAPIKey(ctx context.Context, key *model.APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, key_hash, key_prefix, scopes, rate_limit_tier, name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.pool.Exec(ctx, query,
		key.ID,
		key.UserID,
		key.KeyHash,
		key.KeyPrefix,
		pq.Array(key.Scopes),
		key.RateLimitTier,
		key.Name,
		key.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetAPIKeyByID retrieves an API key by its ID.
func (r *Repository) GetAPIKeyByID(ctx context.Context, id string) (*model.APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, key_prefix, scopes, rate_limit_tier, name, revoked_at, last_used_at, created_at
		FROM api_keys
		WHERE id = $1
	`

	return r.scanAPIKey(r.pool.QueryRow(ctx, query, id))
}

// GetAPIKeysByPrefix retrieves all active API keys matching a prefix.
// Used during authentication to find candidate keys for verification.
func (r *Repository) GetAPIKeysByPrefix(ctx context.Context, prefix string) ([]*model.APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, key_prefix, scopes, rate_limit_tier, name, revoked_at, last_used_at, created_at
		FROM api_keys
		WHERE key_prefix = $1 AND revoked_at IS NULL
	`

	rows, err := r.pool.Query(ctx, query, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys by prefix: %w", err)
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		key, err := r.scanAPIKeyFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return keys, nil
}

// ListAPIKeysByUserID retrieves all API keys for a user.
func (r *Repository) ListAPIKeysByUserID(ctx context.Context, userID string) ([]*model.APIKey, error) {
	query := `
		SELECT id, user_id, key_hash, key_prefix, scopes, rate_limit_tier, name, revoked_at, last_used_at, created_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		key, err := r.scanAPIKeyFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return keys, nil
}

// RevokeAPIKey revokes an API key by setting revoked_at.
func (r *Repository) RevokeAPIKey(ctx context.Context, id string) error {
	query := `
		UPDATE api_keys
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}

	return nil
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp.
// Should be called asynchronously after successful authentication.
func (r *Repository) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}

	return nil
}

// scanAPIKey scans a single row into an APIKey model.
func (r *Repository) scanAPIKey(row pgx.Row) (*model.APIKey, error) {
	var key model.APIKey
	var scopes []string

	err := row.Scan(
		&key.ID,
		&key.UserID,
		&key.KeyHash,
		&key.KeyPrefix,
		pq.Array(&scopes),
		&key.RateLimitTier,
		&key.Name,
		&key.RevokedAt,
		&key.LastUsedAt,
		&key.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to scan API key: %w", err)
	}

	key.Scopes = scopes
	return &key, nil
}

// scanAPIKeyFromRows scans a row from pgx.Rows into an APIKey model.
func (r *Repository) scanAPIKeyFromRows(rows pgx.Rows) (*model.APIKey, error) {
	var key model.APIKey
	var scopes []string

	err := rows.Scan(
		&key.ID,
		&key.UserID,
		&key.KeyHash,
		&key.KeyPrefix,
		pq.Array(&scopes),
		&key.RateLimitTier,
		&key.Name,
		&key.RevokedAt,
		&key.LastUsedAt,
		&key.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	key.Scopes = scopes
	return &key, nil
}
