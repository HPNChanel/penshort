package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/handler/dto"
	"github.com/penshort/penshort/internal/service"
)

// LinkHandler handles HTTP requests for link operations.
type LinkHandler struct {
	svc    *service.LinkService
	logger *slog.Logger
}

// NewLinkHandler creates a new LinkHandler.
func NewLinkHandler(svc *service.LinkService, logger *slog.Logger) *LinkHandler {
	return &LinkHandler{
		svc:    svc,
		logger: logger,
	}
}

// Create handles POST /api/v1/links.
func (h *LinkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	// Default redirect type
	redirectType := req.RedirectType
	if redirectType == 0 {
		redirectType = 302
	}

	input := service.CreateLinkInput{
		Destination:  req.Destination,
		Alias:        req.Alias,
		RedirectType: redirectType,
		ExpiresAt:    req.ExpiresAt,
		OwnerID:      "system", // Phase 2 default
	}

	link, err := h.svc.CreateLink(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.logger.Info("link_created",
		"link_id", link.ID,
		"short_code", link.ShortCode,
		"has_custom_alias", req.Alias != "",
	)

	response := dto.ToLinkResponse(link, h.svc.BaseURL())
	writeJSON(w, http.StatusCreated, response)
}

// Get handles GET /api/v1/links/{id}.
func (h *LinkHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Link ID is required")
		return
	}

	link, err := h.svc.GetLink(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := dto.ToLinkResponse(link, h.svc.BaseURL())
	writeJSON(w, http.StatusOK, response)
}

// List handles GET /api/v1/links.
func (h *LinkHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 20
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	input := service.ListLinksInput{
		OwnerID: "system", // Phase 2 default
		Cursor:  query.Get("cursor"),
		Limit:   limit,
		Status:  query.Get("status"),
	}

	// Parse date filters
	if after := query.Get("created_after"); after != "" {
		if t, err := time.Parse(time.RFC3339, after); err == nil {
			input.CreatedAfter = &t
		}
	}
	if before := query.Get("created_before"); before != "" {
		if t, err := time.Parse(time.RFC3339, before); err == nil {
			input.CreatedBefore = &t
		}
	}

	result, err := h.svc.ListLinks(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	response := dto.ToLinkListResponse(result.Links, h.svc.BaseURL(), result.NextCursor, result.HasMore)
	writeJSON(w, http.StatusOK, response)
}

// Update handles PATCH /api/v1/links/{id}.
func (h *LinkHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Link ID is required")
		return
	}

	var req dto.UpdateLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid request body")
		return
	}

	input := service.UpdateLinkInput{
		ID:          id,
		Destination: req.Destination,
		ExpiresAt:   req.ExpiresAt,
		Enabled:     req.Enabled,
	}

	if req.RedirectType != nil {
		input.RedirectType = req.RedirectType
	}

	link, err := h.svc.UpdateLink(r.Context(), input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.logger.Info("link_updated",
		"link_id", link.ID,
		"short_code", link.ShortCode,
	)

	response := dto.ToLinkResponse(link, h.svc.BaseURL())
	writeJSON(w, http.StatusOK, response)
}

// Delete handles DELETE /api/v1/links/{id}.
func (h *LinkHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_ID", "Link ID is required")
		return
	}

	if err := h.svc.DeleteLink(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.logger.Info("link_deleted", "link_id", id)

	w.WriteHeader(http.StatusNoContent)
}

// handleServiceError maps service errors to HTTP responses.
func (h *LinkHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrLinkNotFound):
		h.writeError(w, http.StatusNotFound, "LINK_NOT_FOUND", "Link not found")
	case errors.Is(err, service.ErrAliasExists):
		h.writeError(w, http.StatusConflict, "ALIAS_TAKEN", "Alias already exists")
	case errors.Is(err, service.ErrInvalidDestination):
		h.writeError(w, http.StatusBadRequest, "INVALID_DESTINATION", "Invalid destination URL")
	case errors.Is(err, service.ErrInvalidAlias):
		h.writeError(w, http.StatusBadRequest, "INVALID_ALIAS", "Invalid alias format")
	case errors.Is(err, service.ErrExpiresInPast):
		h.writeError(w, http.StatusUnprocessableEntity, "EXPIRES_IN_PAST", "Expiry date must be in the future")
	case errors.Is(err, service.ErrInvalidRedirectType):
		h.writeError(w, http.StatusBadRequest, "INVALID_REDIRECT_TYPE", "Redirect type must be 301 or 302")
	case errors.Is(err, service.ErrLinkExpired):
		h.writeError(w, http.StatusConflict, "LINK_EXPIRED", "Cannot update expired link")
	case errors.Is(err, service.ErrURLTooLong):
		h.writeError(w, http.StatusBadRequest, "URL_TOO_LONG", "Destination URL exceeds maximum length")
	default:
		h.logger.Error("internal_error", "error", err)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred")
	}
}

// writeError writes an error response.
func (h *LinkHandler) writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, dto.ErrorResponse{
		Error: message,
		Code:  code,
	})
}
