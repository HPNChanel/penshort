package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/analytics"
	"github.com/penshort/penshort/internal/handler/dto"
	"github.com/penshort/penshort/internal/service"
)

// RedirectHandler handles redirect requests.
type RedirectHandler struct {
	svc       *service.LinkService
	publisher *analytics.Publisher
	logger    *slog.Logger
}

// NewRedirectHandler creates a new RedirectHandler.
func NewRedirectHandler(svc *service.LinkService, publisher *analytics.Publisher, logger *slog.Logger) *RedirectHandler {
	return &RedirectHandler{
		svc:       svc,
		publisher: publisher,
		logger:    logger,
	}
}

// Redirect handles GET /{short_code} for URL redirection.
func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		h.writeError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
		return
	}

	start := time.Now()

	link, cacheHit, err := h.svc.ResolveRedirect(r.Context(), shortCode)
	duration := time.Since(start)

	if err != nil {
		h.handleRedirectError(w, r, shortCode, err, duration)
		return
	}

	// Increment click counter asynchronously
	h.svc.IncrementClickAsync(r.Context(), shortCode)

	// Publish analytics event asynchronously (fire-and-forget)
	if h.publisher != nil {
		clickedAt := time.Now()
		event := analytics.ClickEventPayload{
			ShortCode:   shortCode,
			LinkID:      link.ID,
			OwnerID:     link.OwnerID,
			Referrer:    analytics.SanitizeReferrer(r.Header.Get("Referer")),
			UserAgent:   analytics.TruncateUserAgent(r.Header.Get("User-Agent")),
			VisitorHash: analytics.GenerateVisitorHash(getClientIP(r), r.Header.Get("User-Agent"), clickedAt),
			CountryCode: analytics.ExtractCountryCode(r.Header.Get("CF-IPCountry")),
			ClickedAt:   clickedAt.UnixMilli(),
		}
		h.publisher.PublishAsync(event)
	}

	// Log successful redirect
	h.logger.Info("redirect_success",
		"short_code", shortCode,
		"redirect_type", link.RedirectType,
		"cache_hit", cacheHit,
		"duration_ms", float64(duration.Microseconds())/1000,
	)

	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Cache-Control", "private, max-age=0")

	// Perform redirect
	http.Redirect(w, r, link.Destination, int(link.RedirectType))
}

// handleRedirectError handles errors during redirect resolution.
func (h *RedirectHandler) handleRedirectError(w http.ResponseWriter, r *http.Request, shortCode string, err error, duration time.Duration) {
	switch {
	case errors.Is(err, service.ErrLinkNotFound):
		h.logger.Info("redirect_not_found",
			"short_code", shortCode,
			"duration_ms", float64(duration.Microseconds())/1000,
		)
		h.writeError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Link not found")

	case errors.Is(err, service.ErrLinkExpired):
		h.logger.Info("redirect_expired",
			"short_code", shortCode,
			"reason", "expired",
			"duration_ms", float64(duration.Microseconds())/1000,
		)
		h.writeError(w, http.StatusGone, "LINK_EXPIRED", "Link has expired")

	case errors.Is(err, service.ErrLinkDisabled):
		h.logger.Info("redirect_disabled",
			"short_code", shortCode,
			"reason", "disabled",
			"duration_ms", float64(duration.Microseconds())/1000,
		)
		// Return 404 for disabled links (don't reveal existence)
		h.writeError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Link not found")

	default:
		h.logger.Error("redirect_error",
			"short_code", shortCode,
			"error", err,
			"duration_ms", float64(duration.Microseconds())/1000,
		)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
	}
}

// writeError writes a JSON error response for redirect failures.
func (h *RedirectHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	// Set security headers even on errors
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=0")

	writeJSON(w, status, dto.ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check Cloudflare header first
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	// Check X-Forwarded-For
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Take the first IP in the chain
		for i := 0; i < len(ip); i++ {
			if ip[i] == ',' {
				return ip[:i]
			}
		}
		return ip
	}
	// Check X-Real-IP
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
