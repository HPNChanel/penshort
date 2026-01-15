// Package model defines domain entities for the application.
package model

import "time"

// ClickEvent represents a single click/redirect event.
type ClickEvent struct {
	ID      string `json:"id"`       // ULID (time-sortable)
	EventID string `json:"event_id"` // Idempotency key (Redis stream ID)

	// Link reference
	ShortCode string `json:"short_code"` // Link short code
	LinkID    string `json:"link_id"`    // FK to links.id
	OwnerID   string `json:"owner_id,omitempty"` // Link owner id (not persisted)

	// Request metadata
	Referrer  string `json:"referrer,omitempty"`   // Referer header (truncated 500 chars)
	UserAgent string `json:"user_agent,omitempty"` // UA string (truncated 500 chars)

	// Privacy-safe visitor identification
	VisitorHash string `json:"visitor_hash"` // SHA256(IP + UA + daily_salt)[0:16]

	// Optional geo (from CF-IPCountry header)
	CountryCode string `json:"country_code,omitempty"` // ISO 3166-1 alpha-2

	// Timestamps
	ClickedAt time.Time `json:"clicked_at"` // Event timestamp
	CreatedAt time.Time `json:"created_at"` // DB insertion time
}

// DailyLinkStats represents pre-aggregated daily statistics for a link.
type DailyLinkStats struct {
	ID     string    `json:"id"`      // Composite: link_id:date
	LinkID string    `json:"link_id"` // FK to links.id
	Date   time.Time `json:"date"`    // UTC date (time component zeroed)

	// Counters
	TotalClicks    int64 `json:"total_clicks"`
	UniqueVisitors int64 `json:"unique_visitors"`

	// Breakdowns (stored as JSONB in Postgres)
	ReferrerBreakdown  map[string]int64 `json:"referrer_breakdown,omitempty"`
	UAFamilyBreakdown  map[string]int64 `json:"ua_family_breakdown,omitempty"`
	CountryBreakdown   map[string]int64 `json:"country_breakdown,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AnalyticsSummary represents aggregated analytics for API response.
type AnalyticsSummary struct {
	TotalClicks     int64   `json:"total_clicks"`
	UniqueVisitors  int64   `json:"unique_visitors"`
	AvgClicksPerDay float64 `json:"avg_clicks_per_day"`
}

// AnalyticsResponse represents the full analytics API response.
type AnalyticsResponse struct {
	LinkID    string `json:"link_id"`
	ShortCode string `json:"short_code"`
	Period    struct {
		From string `json:"from"` // ISO date
		To   string `json:"to"`   // ISO date
	} `json:"period"`
	Summary   AnalyticsSummary `json:"summary"`
	Breakdown struct {
		Daily     []DailyBreakdown    `json:"daily,omitempty"`
		Referrers []ReferrerBreakdown `json:"referrers,omitempty"`
		Countries []CountryBreakdown  `json:"countries,omitempty"`
	} `json:"breakdown"`
	GeneratedAt time.Time `json:"generated_at"`
}

// DailyBreakdown represents clicks for a single day.
type DailyBreakdown struct {
	Date           string `json:"date"` // ISO date
	TotalClicks    int64  `json:"total_clicks"`
	UniqueVisitors int64  `json:"unique_visitors"`
}

// ReferrerBreakdown represents clicks from a referrer domain.
type ReferrerBreakdown struct {
	Domain string `json:"domain"`
	Clicks int64  `json:"clicks"`
}

// CountryBreakdown represents clicks from a country.
type CountryBreakdown struct {
	Code   string `json:"code"` // ISO 3166-1 alpha-2
	Name   string `json:"name"` // Full country name
	Clicks int64  `json:"clicks"`
}
