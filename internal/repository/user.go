package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/penshort/penshort/internal/model"
	"github.com/jackc/pgx/v5"
)

// Common errors for user repository operations.
var (
	ErrUserNotFound = errors.New("user not found")
	ErrEmailExists  = errors.New("email already exists")
)

// CreateUser inserts a new user into the database.
func (r *Repository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, created_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			return ErrEmailExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by their ID.
func (r *Repository) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, created_at
		FROM users
		WHERE id = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by their email address.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, created_at
		FROM users
		WHERE email = $1
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetOrCreateUser gets a user by email or creates one if not found.
func (r *Repository) GetOrCreateUser(ctx context.Context, user *model.User) (*model.User, error) {
	existing, err := r.GetUserByEmail(ctx, user.Email)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Create new user
	user.CreatedAt = time.Now()
	if err := r.CreateUser(ctx, user); err != nil {
		// Handle race condition - another request may have created it
		if errors.Is(err, ErrEmailExists) {
			return r.GetUserByEmail(ctx, user.Email)
		}
		return nil, err
	}

	return user, nil
}
