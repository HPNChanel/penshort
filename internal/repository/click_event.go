package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/penshort/penshort/internal/model"
)

// ClickEventRepository provides database access for click events.
type ClickEventRepository struct {
	repo *Repository
}

// NewClickEventRepository creates a new ClickEventRepository.
func NewClickEventRepository(repo *Repository) *ClickEventRepository {
	return &ClickEventRepository{repo: repo}
}

// BulkInsert inserts multiple click events with idempotency via ON CONFLICT DO NOTHING.
func (r *ClickEventRepository) BulkInsert(ctx context.Context, events []*model.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use COPY for large batches, but for moderate sizes (< 1000), use multi-row INSERT
	batch := &pgx.Batch{}

	query := `
		INSERT INTO click_events (
			id, event_id, short_code, link_id, referrer, user_agent,
			visitor_hash, country_code, clicked_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (event_id) DO NOTHING
	`

	for _, event := range events {
		batch.Queue(query,
			event.ID,
			event.EventID,
			event.ShortCode,
			event.LinkID,
			nullableString(event.Referrer),
			nullableString(event.UserAgent),
			event.VisitorHash,
			nullableString(event.CountryCode),
			event.ClickedAt,
		)
	}

	results := r.repo.pool.SendBatch(ctx, batch)
	defer results.Close()

	// Check for errors in batch execution
	for i := 0; i < len(events); i++ {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("batch insert event %d: %w", i, err)
		}
	}

	return nil
}

// UpdateDailyStats updates the daily_link_stats table with aggregated data.
func (r *ClickEventRepository) UpdateDailyStats(ctx context.Context, events []*model.ClickEvent) error {
	if len(events) == 0 {
		return nil
	}

	keys := uniqueDailyKeys(events)
	for _, key := range keys {
		acc, err := r.recalculateDailyStat(ctx, key.linkID, key.date)
		if err != nil {
			return fmt.Errorf("recalculate daily stat %s:%s: %w", key.linkID, key.date.Format("2006-01-02"), err)
		}
		if err := r.upsertDailyStat(ctx, acc); err != nil {
			return fmt.Errorf("upsert daily stat %s:%s: %w", key.linkID, key.date.Format("2006-01-02"), err)
		}
	}

	return nil
}

// dailyStatsAccumulator accumulates stats for a single link/date combination.
type dailyStatsAccumulator struct {
	linkID         string
	date           time.Time
	totalClicks    int64
	uniqueVisitors int64
	referrers      map[string]int64
	countries      map[string]int64
	visitorSeen    map[string]bool
}

type dailyStatsKey struct {
	linkID string
	date   time.Time
}

func uniqueDailyKeys(events []*model.ClickEvent) []dailyStatsKey {
	seen := make(map[string]dailyStatsKey)
	for _, event := range events {
		day := event.ClickedAt.UTC().Truncate(24 * time.Hour)
		key := fmt.Sprintf("%s:%s", event.LinkID, day.Format("2006-01-02"))
		seen[key] = dailyStatsKey{linkID: event.LinkID, date: day}
	}

	keys := make([]dailyStatsKey, 0, len(seen))
	for _, key := range seen {
		keys = append(keys, key)
	}
	return keys
}

func (r *ClickEventRepository) recalculateDailyStat(ctx context.Context, linkID string, date time.Time) (*dailyStatsAccumulator, error) {
	start := date.UTC().Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	query := `
		SELECT COALESCE(referrer, ''), COALESCE(country_code, ''), visitor_hash
		FROM click_events
		WHERE link_id = $1 AND clicked_at >= $2 AND clicked_at < $3
	`

	rows, err := r.repo.pool.Query(ctx, query, linkID, start, end)
	if err != nil {
		return nil, fmt.Errorf("query click events: %w", err)
	}
	defer rows.Close()

	events := make([]*model.ClickEvent, 0)
	for rows.Next() {
		var referrer, country, visitorHash string
		if err := rows.Scan(&referrer, &country, &visitorHash); err != nil {
			return nil, fmt.Errorf("scan click event: %w", err)
		}
		events = append(events, &model.ClickEvent{
			Referrer:    referrer,
			CountryCode: country,
			VisitorHash: visitorHash,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate click events: %w", err)
	}

	acc := accumulateDailyStats(events)
	acc.linkID = linkID
	acc.date = start
	return acc, nil
}

func accumulateDailyStats(events []*model.ClickEvent) *dailyStatsAccumulator {
	acc := &dailyStatsAccumulator{
		referrers:   make(map[string]int64),
		countries:   make(map[string]int64),
		visitorSeen: make(map[string]bool),
	}

	for _, event := range events {
		acc.totalClicks++

		if event.VisitorHash != "" && !acc.visitorSeen[event.VisitorHash] {
			acc.visitorSeen[event.VisitorHash] = true
			acc.uniqueVisitors++
		}

		if event.Referrer != "" {
			domain := extractDomain(event.Referrer)
			acc.referrers[domain]++
		} else {
			acc.referrers["(direct)"]++
		}

		if event.CountryCode != "" {
			acc.countries[event.CountryCode]++
		}
	}

	return acc
}

// upsertDailyStat inserts or updates a daily_link_stats row.
func (r *ClickEventRepository) upsertDailyStat(ctx context.Context, acc *dailyStatsAccumulator) error {
	referrerJSON, _ := json.Marshal(acc.referrers)
	countryJSON, _ := json.Marshal(acc.countries)
	id := fmt.Sprintf("%s:%s", acc.linkID, acc.date.Format("2006-01-02"))

	query := `
		INSERT INTO daily_link_stats (
			id, link_id, date, total_clicks, unique_visitors,
			referrer_breakdown, country_breakdown, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (link_id, date) DO UPDATE SET
			total_clicks = EXCLUDED.total_clicks,
			unique_visitors = EXCLUDED.unique_visitors,
			referrer_breakdown = EXCLUDED.referrer_breakdown,
			country_breakdown = EXCLUDED.country_breakdown,
			updated_at = NOW()
	`

	_, err := r.repo.pool.Exec(ctx, query,
		id,
		acc.linkID,
		acc.date,
		acc.totalClicks,
		acc.uniqueVisitors,
		referrerJSON,
		countryJSON,
	)

	return err
}

// GetDailyStats retrieves daily stats for a link within a date range.
func (r *ClickEventRepository) GetDailyStats(ctx context.Context, linkID string, from, to time.Time) ([]*model.DailyLinkStats, error) {
	query := `
		SELECT id, link_id, date, total_clicks, unique_visitors,
			   referrer_breakdown, ua_family_breakdown, country_breakdown,
			   created_at, updated_at
		FROM daily_link_stats
		WHERE link_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC
	`

	rows, err := r.repo.pool.Query(ctx, query, linkID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*model.DailyLinkStats
	for rows.Next() {
		stat, err := r.scanDailyStat(rows)
		if err != nil {
			return nil, fmt.Errorf("scan daily stat: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

// GetAnalyticsSummary retrieves aggregated analytics for a link.
func (r *ClickEventRepository) GetAnalyticsSummary(ctx context.Context, linkID string, from, to time.Time) (*model.AnalyticsSummary, error) {
	query := `
		SELECT 
			COALESCE(SUM(total_clicks), 0) as total_clicks,
			COALESCE(SUM(unique_visitors), 0) as unique_visitors,
			COUNT(*) as days
		FROM daily_link_stats
		WHERE link_id = $1 AND date >= $2 AND date <= $3
	`

	var totalClicks, uniqueVisitors int64
	var days int

	err := r.repo.pool.QueryRow(ctx, query, linkID, from, to).Scan(&totalClicks, &uniqueVisitors, &days)
	if err != nil {
		return nil, fmt.Errorf("query analytics summary: %w", err)
	}

	var avgClicksPerDay float64
	if days > 0 {
		avgClicksPerDay = float64(totalClicks) / float64(days)
	}

	return &model.AnalyticsSummary{
		TotalClicks:     totalClicks,
		UniqueVisitors:  uniqueVisitors,
		AvgClicksPerDay: avgClicksPerDay,
	}, nil
}

// GetTopReferrers returns the top referrer domains for a link.
func (r *ClickEventRepository) GetTopReferrers(ctx context.Context, linkID string, from, to time.Time, limit int) ([]model.ReferrerBreakdown, error) {
	// Aggregate JSONB from daily stats
	query := `
		WITH aggregated AS (
			SELECT jsonb_object_agg(key, value::bigint) as combined
			FROM daily_link_stats, jsonb_each_text(referrer_breakdown)
			WHERE link_id = $1 AND date >= $2 AND date <= $3
		)
		SELECT key as domain, SUM(value::bigint) as clicks
		FROM daily_link_stats, jsonb_each_text(referrer_breakdown)
		WHERE link_id = $1 AND date >= $2 AND date <= $3
		GROUP BY key
		ORDER BY clicks DESC
		LIMIT $4
	`

	rows, err := r.repo.pool.Query(ctx, query, linkID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query top referrers: %w", err)
	}
	defer rows.Close()

	var referrers []model.ReferrerBreakdown
	for rows.Next() {
		var r model.ReferrerBreakdown
		if err := rows.Scan(&r.Domain, &r.Clicks); err != nil {
			return nil, fmt.Errorf("scan referrer: %w", err)
		}
		referrers = append(referrers, r)
	}

	return referrers, rows.Err()
}

// GetTopCountries returns the top countries for a link.
func (r *ClickEventRepository) GetTopCountries(ctx context.Context, linkID string, from, to time.Time, limit int) ([]model.CountryBreakdown, error) {
	query := `
		SELECT key as code, SUM(value::bigint) as clicks
		FROM daily_link_stats, jsonb_each_text(country_breakdown)
		WHERE link_id = $1 AND date >= $2 AND date <= $3
		GROUP BY key
		ORDER BY clicks DESC
		LIMIT $4
	`

	rows, err := r.repo.pool.Query(ctx, query, linkID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query top countries: %w", err)
	}
	defer rows.Close()

	var countries []model.CountryBreakdown
	for rows.Next() {
		var c model.CountryBreakdown
		if err := rows.Scan(&c.Code, &c.Clicks); err != nil {
			return nil, fmt.Errorf("scan country: %w", err)
		}
		c.Name = countryName(c.Code)
		countries = append(countries, c)
	}

	return countries, rows.Err()
}

// scanDailyStat scans a row into DailyLinkStats.
func (r *ClickEventRepository) scanDailyStat(rows pgx.Rows) (*model.DailyLinkStats, error) {
	var stat model.DailyLinkStats
	var referrerJSON, uaJSON, countryJSON []byte

	err := rows.Scan(
		&stat.ID,
		&stat.LinkID,
		&stat.Date,
		&stat.TotalClicks,
		&stat.UniqueVisitors,
		&referrerJSON,
		&uaJSON,
		&countryJSON,
		&stat.CreatedAt,
		&stat.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSONB fields
	if len(referrerJSON) > 0 {
		_ = json.Unmarshal(referrerJSON, &stat.ReferrerBreakdown)
	}
	if len(uaJSON) > 0 {
		_ = json.Unmarshal(uaJSON, &stat.UAFamilyBreakdown)
	}
	if len(countryJSON) > 0 {
		_ = json.Unmarshal(countryJSON, &stat.CountryBreakdown)
	}

	return &stat, nil
}

// nullableString returns nil for empty strings.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// extractDomain extracts domain from URL.
func extractDomain(urlStr string) string {
	// Simple extraction - in production use net/url
	if len(urlStr) < 10 {
		return "(unknown)"
	}
	// Find :// and next /
	start := 0
	for i := 0; i < len(urlStr)-2; i++ {
		if urlStr[i:i+3] == "://" {
			start = i + 3
			break
		}
	}
	end := len(urlStr)
	for i := start; i < len(urlStr); i++ {
		if urlStr[i] == '/' {
			end = i
			break
		}
	}
	if start >= end {
		return "(unknown)"
	}
	return urlStr[start:end]
}

// countryName returns the full name for a country code.
func countryName(code string) string {
	// Simplified mapping - in production use a proper library
	names := map[string]string{
		"US": "United States",
		"VN": "Vietnam",
		"GB": "United Kingdom",
		"DE": "Germany",
		"FR": "France",
		"JP": "Japan",
		"CN": "China",
		"KR": "South Korea",
		"IN": "India",
		"BR": "Brazil",
		"CA": "Canada",
		"AU": "Australia",
		"SG": "Singapore",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
