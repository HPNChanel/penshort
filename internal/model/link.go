// Package model defines domain entities for the application.
package model

import (
	"strconv"
	"time"
)

// LinkStatus represents the computed status of a link.
type LinkStatus string

const (
	LinkStatusActive   LinkStatus = "active"
	LinkStatusExpired  LinkStatus = "expired"
	LinkStatusDisabled LinkStatus = "disabled"
	LinkStatusDeleted  LinkStatus = "deleted"
)

// RedirectType represents the HTTP redirect status code.
type RedirectType int

const (
	RedirectPermanent RedirectType = 301
	RedirectTemporary RedirectType = 302
)

// IsValid checks if the redirect type is valid.
func (r RedirectType) IsValid() bool {
	return r == RedirectPermanent || r == RedirectTemporary
}

// Link represents a shortened URL entity.
type Link struct {
	ID           string       `json:"id"`
	ShortCode    string       `json:"short_code"`
	Destination  string       `json:"destination"`
	RedirectType RedirectType `json:"redirect_type"`
	OwnerID      string       `json:"owner_id"`
	Enabled      bool         `json:"enabled"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
	DeletedAt    *time.Time   `json:"-"`
	ClickCount   int64        `json:"click_count"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Status computes the current status of the link.
func (l *Link) Status() LinkStatus {
	if l.DeletedAt != nil {
		return LinkStatusDeleted
	}
	if !l.Enabled {
		return LinkStatusDisabled
	}
	if l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt) {
		return LinkStatusExpired
	}
	return LinkStatusActive
}

// IsActive returns true if the link can be used for redirects.
func (l *Link) IsActive() bool {
	return l.Status() == LinkStatusActive
}

// IsExpired returns true if the link has time-based expiry.
func (l *Link) IsExpired() bool {
	return l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt)
}

// CachedLink represents link data stored in Redis cache.
// Uses string types for Redis hash compatibility.
type CachedLink struct {
	Destination  string `redis:"destination"`
	RedirectType string `redis:"redirect_type"`
	ExpiresAt    string `redis:"expires_at"`  // Unix timestamp or empty
	Enabled      string `redis:"enabled"`     // "1" or "0"
	DeletedAt    string `redis:"deleted_at"`  // Unix timestamp or empty
	UpdatedAt    string `redis:"updated_at"`  // Unix timestamp
}

// ToLink converts CachedLink to Link domain model.
func (c *CachedLink) ToLink(shortCode string) *Link {
	link := &Link{
		ShortCode:   shortCode,
		Destination: c.Destination,
		Enabled:     c.Enabled == "1",
	}

	// Parse redirect type
	if c.RedirectType == "301" {
		link.RedirectType = RedirectPermanent
	} else {
		link.RedirectType = RedirectTemporary
	}

	// Parse expires_at
	if c.ExpiresAt != "" {
		if ts, err := strconv.ParseInt(c.ExpiresAt, 10, 64); err == nil {
			t := time.Unix(ts, 0)
			link.ExpiresAt = &t
		}
	}

	// Parse deleted_at
	if c.DeletedAt != "" {
		if ts, err := strconv.ParseInt(c.DeletedAt, 10, 64); err == nil {
			t := time.Unix(ts, 0)
			link.DeletedAt = &t
		}
	}

	// Parse updated_at
	if c.UpdatedAt != "" {
		if ts, err := strconv.ParseInt(c.UpdatedAt, 10, 64); err == nil {
			link.UpdatedAt = time.Unix(ts, 0)
		}
	}

	return link
}

// ToCachedLink converts Link domain model to CachedLink.
func (l *Link) ToCachedLink() *CachedLink {
	cached := &CachedLink{
		Destination:  l.Destination,
		RedirectType: strconv.Itoa(int(l.RedirectType)),
		Enabled:      boolToString(l.Enabled),
		UpdatedAt:    strconv.FormatInt(l.UpdatedAt.Unix(), 10),
	}

	if l.ExpiresAt != nil {
		cached.ExpiresAt = strconv.FormatInt(l.ExpiresAt.Unix(), 10)
	}

	if l.DeletedAt != nil {
		cached.DeletedAt = strconv.FormatInt(l.DeletedAt.Unix(), 10)
	}

	return cached
}

// boolToString converts boolean to "1" or "0".
func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
