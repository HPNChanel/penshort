package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/penshort/penshort/internal/model"
)

// AdminLinkSearcher defines the interface for link search operations.
type AdminLinkSearcher interface {
	GetLinkByShortCode(ctx context.Context, shortCode string) (*model.Link, error)
	SearchLinksByDestination(ctx context.Context, destination string, limit int) ([]*model.Link, error)
}

// AdminKeyLister defines the interface for listing API keys.
type AdminKeyLister interface {
	ListAPIKeysByUserID(ctx context.Context, userID string) ([]*model.APIKey, error)
}

// AdminHandler provides admin-only endpoints for debugging and operations.
type AdminHandler struct {
	linkRepo   AdminLinkSearcher
	keyRepo    AdminKeyLister
	logger     *slog.Logger
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(linkRepo AdminLinkSearcher, keyRepo AdminKeyLister, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		linkRepo:   linkRepo,
		keyRepo:    keyRepo,
		logger:     logger,
	}
}

// LinkLookupResponse represents the response for link lookup.
type LinkLookupResponse struct {
	Links []AdminLinkResponse `json:"links"`
	Total int                 `json:"total"`
}

// AdminLinkResponse represents a link in admin context with extended info.
type AdminLinkResponse struct {
	ID           string              `json:"id"`
	ShortCode    string              `json:"short_code"`
	Destination  string              `json:"destination"`
	RedirectType model.RedirectType  `json:"redirect_type"`
	OwnerID      string              `json:"owner_id"`
	Enabled      bool                `json:"enabled"`
	ClickCount   int64               `json:"click_count"`
	ExpiresAt    *time.Time          `json:"expires_at,omitempty"`
	DeletedAt    *time.Time          `json:"deleted_at,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// LookupLinks handles GET /api/v1/admin/links?q={shortcode|destination}
// Searches by short code (exact match) or destination URL (partial match).
func (h *AdminHandler) LookupLinks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeErrorJSON(w, http.StatusBadRequest, "MISSING_QUERY", "query parameter 'q' is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var links []*model.Link

	// Try exact short code lookup first
	if link, err := h.linkRepo.GetLinkByShortCode(ctx, query); err == nil {
		links = append(links, link)
	}

	// If no exact match and query looks like a URL, search by destination
	if len(links) == 0 && (len(query) > 10 || containsScheme(query)) {
		destLinks, err := h.linkRepo.SearchLinksByDestination(ctx, query, 20)
		if err != nil {
			h.logger.Error("failed to search links by destination",
				"error", err,
				"query", truncateForLog(query, 100),
			)
		} else {
			links = destLinks
		}
	}

	response := LinkLookupResponse{
		Links: make([]AdminLinkResponse, 0, len(links)),
		Total: len(links),
	}

	for _, link := range links {
		response.Links = append(response.Links, AdminLinkResponse{
			ID:           link.ID,
			ShortCode:    link.ShortCode,
			Destination:  link.Destination,
			RedirectType: link.RedirectType,
			OwnerID:      link.OwnerID,
			Enabled:      link.Enabled,
			ClickCount:   link.ClickCount,
			ExpiresAt:    link.ExpiresAt,
			DeletedAt:    link.DeletedAt,
			CreatedAt:    link.CreatedAt,
			UpdatedAt:    link.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, response)
}

// AdminAPIKeyListResponse represents the response for API key listing.
type AdminAPIKeyListResponse struct {
	Keys  []model.APIKeyResponse `json:"keys"`
	Total int                    `json:"total"`
}

// ListAPIKeysByUser handles GET /api/v1/admin/api-keys?user_id={id}
// Lists all API keys for a specific user (admin only).
func (h *AdminHandler) ListAPIKeysByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "MISSING_USER_ID", "query parameter 'user_id' is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	keys, err := h.keyRepo.ListAPIKeysByUserID(ctx, userID)
	if err != nil {
		h.logger.Error("failed to list API keys",
			"error", err,
			"user_id", userID,
		)
		writeErrorJSON(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list API keys")
		return
	}

	response := AdminAPIKeyListResponse{
		Keys:  make([]model.APIKeyResponse, 0, len(keys)),
		Total: len(keys),
	}

	for _, key := range keys {
		response.Keys = append(response.Keys, key.ToResponse())
	}

	writeJSON(w, http.StatusOK, response)
}

// StatsResponse represents operational statistics.
type StatsResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime,omitempty"`
}

// Stats handles GET /api/v1/admin/stats
// Returns basic operational statistics.
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	response := StatsResponse{
		Timestamp: time.Now().UTC(),
		Service:   "penshort",
		Version:   "1.0.0", // TODO: inject at build time
	}
	writeJSON(w, http.StatusOK, response)
}

// containsScheme checks if a string looks like a URL.
func containsScheme(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}

// truncateForLog truncates a string for logging purposes.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// writeErrorJSON writes a JSON error response.
func writeErrorJSON(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}
