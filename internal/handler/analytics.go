package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/handler/dto"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
)

// AnalyticsHandler handles analytics API requests.
type AnalyticsHandler struct {
	repo   *repository.ClickEventRepository
	logger *slog.Logger
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(repo *repository.ClickEventRepository, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		repo:   repo,
		logger: logger.With("component", "handler.analytics"),
	}
}

// GetLinkAnalytics handles GET /v1/links/{id}/analytics.
func (h *AnalyticsHandler) GetLinkAnalytics(w http.ResponseWriter, r *http.Request) {
	linkID := chi.URLParam(r, "id")
	if linkID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Link ID is required")
		return
	}

	// Parse query parameters
	from, to := h.parseTimeRange(r)
	includes := h.parseIncludes(r)

	// Get summary
	summary, err := h.repo.GetAnalyticsSummary(r.Context(), linkID, from, to)
	if err != nil {
		h.logger.Error("failed to get analytics summary", "link_id", linkID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch analytics")
		return
	}

	// Get daily breakdown
	dailyStats, err := h.repo.GetDailyStats(r.Context(), linkID, from, to)
	if err != nil {
		h.logger.Error("failed to get daily stats", "link_id", linkID, "error", err)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch analytics")
		return
	}

	// Build response
	response := h.buildAnalyticsResponse(linkID, from, to, summary, dailyStats, includes, r.Context())

	writeJSON(w, http.StatusOK, response)
}

// parseTimeRange extracts from/to dates from query params.
func (h *AnalyticsHandler) parseTimeRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now().UTC()
	defaultFrom := now.AddDate(0, 0, -7) // 7 days ago
	defaultTo := now

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from := defaultFrom
	to := defaultTo

	if fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = parsed
		}
	}

	if toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			to = parsed
		}
	}

	// Cap to 90 days max
	if to.Sub(from) > 90*24*time.Hour {
		from = to.AddDate(0, 0, -90)
	}

	// Don't allow future dates
	if to.After(now) {
		to = now
	}

	return from, to
}

// parseIncludes extracts included breakdown types from query.
func (h *AnalyticsHandler) parseIncludes(r *http.Request) map[string]bool {
	includes := make(map[string]bool)
	includeStr := r.URL.Query().Get("include")

	if includeStr == "" {
		// Default: include all
		includes["referrers"] = true
		includes["countries"] = true
		includes["daily"] = true
		return includes
	}

	for _, inc := range splitComma(includeStr) {
		includes[inc] = true
	}

	return includes
}

// buildAnalyticsResponse constructs the API response.
func (h *AnalyticsHandler) buildAnalyticsResponse(
	linkID string,
	from, to time.Time,
	summary *model.AnalyticsSummary,
	dailyStats []*model.DailyLinkStats,
	includes map[string]bool,
	ctx interface{},
) *model.AnalyticsResponse {
	response := &model.AnalyticsResponse{
		LinkID:      linkID,
		GeneratedAt: time.Now().UTC(),
	}
	response.Period.From = from.Format("2006-01-02")
	response.Period.To = to.Format("2006-01-02")
	response.Summary = *summary

	// Daily breakdown
	if includes["daily"] {
		for _, stat := range dailyStats {
			response.Breakdown.Daily = append(response.Breakdown.Daily, model.DailyBreakdown{
				Date:           stat.Date.Format("2006-01-02"),
				TotalClicks:    stat.TotalClicks,
				UniqueVisitors: stat.UniqueVisitors,
			})
		}
	}

	// Aggregate referrers from daily stats
	if includes["referrers"] {
		referrerTotals := make(map[string]int64)
		for _, stat := range dailyStats {
			for domain, count := range stat.ReferrerBreakdown {
				referrerTotals[domain] += count
			}
		}
		response.Breakdown.Referrers = sortedBreakdown(referrerTotals, 10)
	}

	// Aggregate countries from daily stats
	if includes["countries"] {
		countryTotals := make(map[string]int64)
		for _, stat := range dailyStats {
			for code, count := range stat.CountryBreakdown {
				countryTotals[code] += count
			}
		}
		response.Breakdown.Countries = sortedCountryBreakdown(countryTotals, 10)
	}

	return response
}

// writeError writes a JSON error response.
func (h *AnalyticsHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, dto.ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// sortedBreakdown converts map to sorted slice of ReferrerBreakdown.
func sortedBreakdown(m map[string]int64, limit int) []model.ReferrerBreakdown {
	// Simple implementation - in production use a heap for top-k
	result := make([]model.ReferrerBreakdown, 0, len(m))
	for domain, clicks := range m {
		result = append(result, model.ReferrerBreakdown{
			Domain: domain,
			Clicks: clicks,
		})
	}

	// Sort by clicks descending (simple bubble sort for small sets)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Clicks > result[i].Clicks {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > limit {
		return result[:limit]
	}
	return result
}

// sortedCountryBreakdown converts map to sorted slice of CountryBreakdown.
func sortedCountryBreakdown(m map[string]int64, limit int) []model.CountryBreakdown {
	result := make([]model.CountryBreakdown, 0, len(m))
	for code, clicks := range m {
		result = append(result, model.CountryBreakdown{
			Code:   code,
			Name:   countryName(code),
			Clicks: clicks,
		})
	}

	// Sort by clicks descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Clicks > result[i].Clicks {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	if len(result) > limit {
		return result[:limit]
	}
	return result
}

// countryName returns full name for country code.
func countryName(code string) string {
	names := map[string]string{
		"US": "United States", "VN": "Vietnam", "GB": "United Kingdom",
		"DE": "Germany", "FR": "France", "JP": "Japan", "CN": "China",
		"KR": "South Korea", "IN": "India", "BR": "Brazil", "CA": "Canada",
		"AU": "Australia", "SG": "Singapore", "TH": "Thailand", "ID": "Indonesia",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}

// splitComma splits a comma-separated string.
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if start < i {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}
