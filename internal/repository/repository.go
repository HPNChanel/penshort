// Package repository provides database access layer.
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides database access methods.
type Repository struct {
	pool *pgxpool.Pool
}

// New creates a new Repository with a connection pool.
func New(ctx context.Context, databaseURL string) (*Repository, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Connection pool settings
	config.MaxConns = 10
	config.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{pool: pool}, nil
}

// Ping checks database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Close closes the database connection pool.
func (r *Repository) Close() {
	r.pool.Close()
}

// Pool returns the underlying connection pool.
// Use sparingly - prefer adding methods to Repository.
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}
