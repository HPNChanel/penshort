// Package model defines domain entities for the application.
package model

import "time"

// User represents a minimal user entity for API key ownership.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
