package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/penshort/penshort/internal/model"
)

// Common errors for link repository operations.
var (
	ErrLinkNotFound     = errors.New("link not found")
	ErrAliasExists      = errors.New("alias already exists")
	ErrLinkExpired      = errors.New("link is expired")
	ErrInvalidCursor    = errors.New("invalid pagination cursor")
)

// LinkFilter defines filters for listing links.
type LinkFilter struct {
	OwnerID       string
	Status        model.LinkStatus
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// PaginationCursor represents decoded cursor for pagination.
type PaginationCursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateLink inserts a new link into the database.
func (r *Repository) CreateLink(ctx context.Context, link *model.Link) error {
	query := `
		INSERT INTO links (id, short_code, destination, redirect_type, owner_id, enabled, expires_at, click_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.pool.Exec(ctx, query,
		link.ID,
		link.ShortCode,
		link.Destination,
		link.RedirectType,
		link.OwnerID,
		link.Enabled,
		link.ExpiresAt,
		link.ClickCount,
		link.CreatedAt,
		link.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return ErrAliasExists
		}
		return fmt.Errorf("failed to create link: %w", err)
	}

	return nil
}

// GetLinkByID retrieves a link by its ID.
func (r *Repository) GetLinkByID(ctx context.Context, id string) (*model.Link, error) {
	query := `
		SELECT id, short_code, destination, redirect_type, owner_id, enabled, expires_at, deleted_at, click_count, created_at, updated_at
		FROM links
		WHERE id = $1 AND deleted_at IS NULL
	`

	link, err := r.scanLink(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("failed to get link by ID: %w", err)
	}

	return link, nil
}

// GetLinkByShortCode retrieves a link by its short code.
// This is the hot path for redirects.
func (r *Repository) GetLinkByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	query := `
		SELECT id, short_code, destination, redirect_type, owner_id, enabled, expires_at, deleted_at, click_count, created_at, updated_at
		FROM links
		WHERE short_code = $1 AND deleted_at IS NULL
	`

	link, err := r.scanLink(r.pool.QueryRow(ctx, query, shortCode))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("failed to get link by short code: %w", err)
	}

	return link, nil
}

// GetLinkByCode retrieves a link by its short code.
// Alias for GetLinkByShortCode to match external naming.
func (r *Repository) GetLinkByCode(ctx context.Context, shortCode string) (*model.Link, error) {
	return r.GetLinkByShortCode(ctx, shortCode)
}

// ListLinks retrieves a paginated list of links.
func (r *Repository) ListLinks(ctx context.Context, filter LinkFilter, cursor string, limit int) ([]*model.Link, string, error) {
	// Decode cursor if provided
	var cursorData *PaginationCursor
	if cursor != "" {
		var err error
		cursorData, err = decodeCursor(cursor)
		if err != nil {
			return nil, "", ErrInvalidCursor
		}
	}

	// Build query with filters
	query := `
		SELECT id, short_code, destination, redirect_type, owner_id, enabled, expires_at, deleted_at, click_count, created_at, updated_at
		FROM links
		WHERE deleted_at IS NULL
		  AND owner_id = $1
	`
	args := []any{filter.OwnerID}
	argIndex := 2

	if cursorData != nil {
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorData.CreatedAt, cursorData.ID)
		argIndex += 2
	}

	if filter.CreatedAfter != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.CreatedAfter)
		argIndex++
	}

	if filter.CreatedBefore != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.CreatedBefore)
		argIndex++
	}

	// Note: Status filtering is computed at app level, not DB level

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argIndex)
	args = append(args, limit+1) // Fetch one extra to determine hasMore

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list links: %w", err)
	}
	defer rows.Close()

	var links []*model.Link
	for rows.Next() {
		link, err := r.scanLinkFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan link: %w", err)
		}
		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating links: %w", err)
	}

	// Determine if there are more results
	var nextCursor string
	if len(links) > limit {
		links = links[:limit] // Remove extra row
		lastLink := links[len(links)-1]
		nextCursor = encodeCursor(&PaginationCursor{
			ID:        lastLink.ID,
			CreatedAt: lastLink.CreatedAt,
		})
	}

	return links, nextCursor, nil
}

// UpdateLink updates a link's mutable fields.
func (r *Repository) UpdateLink(ctx context.Context, link *model.Link) error {
	query := `
		UPDATE links
		SET destination = $2, redirect_type = $3, enabled = $4, expires_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query,
		link.ID,
		link.Destination,
		link.RedirectType,
		link.Enabled,
		link.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update link: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}

	return nil
}

// DeleteLink performs a soft delete on a link.
func (r *Repository) DeleteLink(ctx context.Context, id string) error {
	query := `
		UPDATE links
		SET deleted_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrLinkNotFound
	}

	return nil
}

// IncrementClickCount increments the click counter for a link.
// This is typically called from a background job, not the redirect path.
func (r *Repository) IncrementClickCount(ctx context.Context, id string, count int64) error {
	query := `
		UPDATE links
		SET click_count = click_count + $2
		WHERE id = $1
	`

	_, err := r.pool.Exec(ctx, query, id, count)
	if err != nil {
		return fmt.Errorf("failed to increment click count: %w", err)
	}

	return nil
}

// ShortCodeExists checks if a short code already exists.
func (r *Repository) ShortCodeExists(ctx context.Context, shortCode string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1 AND deleted_at IS NULL)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, shortCode).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check short code existence: %w", err)
	}

	return exists, nil
}

// scanLink scans a single row into a Link model.
func (r *Repository) scanLink(row pgx.Row) (*model.Link, error) {
	var link model.Link
	err := row.Scan(
		&link.ID,
		&link.ShortCode,
		&link.Destination,
		&link.RedirectType,
		&link.OwnerID,
		&link.Enabled,
		&link.ExpiresAt,
		&link.DeletedAt,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
	)
	return &link, err
}

// scanLinkFromRows scans a row from pgx.Rows into a Link model.
func (r *Repository) scanLinkFromRows(rows pgx.Rows) (*model.Link, error) {
	var link model.Link
	err := rows.Scan(
		&link.ID,
		&link.ShortCode,
		&link.Destination,
		&link.RedirectType,
		&link.OwnerID,
		&link.Enabled,
		&link.ExpiresAt,
		&link.DeletedAt,
		&link.ClickCount,
		&link.CreatedAt,
		&link.UpdatedAt,
	)
	return &link, err
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	// PostgreSQL error code 23505 is unique_violation
	return err != nil && (contains(err.Error(), "23505") || contains(err.Error(), "unique"))
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

// searchString is a simple string search.
func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// encodeCursor encodes pagination cursor to base64.
func encodeCursor(cursor *PaginationCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// decodeCursor decodes base64 pagination cursor.
func decodeCursor(s string) (*PaginationCursor, error) {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}

	var cursor PaginationCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}

	return &cursor, nil
}
