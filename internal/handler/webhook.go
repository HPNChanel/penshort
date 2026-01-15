package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/handler/dto"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/webhook"
)

// getAuthContext is a helper to extract auth context from request.
func getAuthContext(ctx context.Context) *model.AuthContext {
	return auth.AuthFromContext(ctx)
}

// WebhookHandler handles webhook management endpoints.
type WebhookHandler struct {
	repo           *webhook.Repository
	logger         *slog.Logger
	allowInsecure  bool
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(repo *webhook.Repository, logger *slog.Logger, allowInsecure bool) *WebhookHandler {
	return &WebhookHandler{
		repo:          repo,
		logger:        logger.With("handler", "webhook"),
		allowInsecure: allowInsecure,
	}
}

// Create handles POST /api/v1/webhooks
func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	// Check scope
	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	var req model.WebhookEndpointCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid request body",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Validate target URL
	if err := webhook.ValidateTargetURLWithOptions(req.TargetURL, webhook.ValidationOptions{AllowInsecure: h.allowInsecure}); err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_URL",
		})
		return
	}

	// Validate event types
	eventTypes := req.EventTypes
	if len(eventTypes) == 0 {
		eventTypes = []model.EventType{model.EventTypeClick}
	}
	for _, et := range eventTypes {
		if !model.IsValidEventType(et) {
			writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
				Error: "Invalid event type: " + string(et),
				Code:  "INVALID_EVENT_TYPE",
			})
			return
		}
	}

	// Generate secret
	secret, err := webhook.GenerateSecret()
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to create webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	now := time.Now()
	endpoint := &model.WebhookEndpoint{
		ID:          generateID(),
		UserID:      auth.UserID,
		TargetURL:   req.TargetURL,
		SecretHash:  webhook.HashSecret(secret),
		Enabled:     true,
		EventTypes:  eventTypes,
		Name:        req.Name,
		Description: req.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.CreateEndpoint(ctx, endpoint); err != nil {
		h.logger.Error("failed to create endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to create webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	h.logger.Info("webhook endpoint created",
		"endpoint_id", endpoint.ID,
		"user_id", auth.UserID,
	)

	// Return with secret (only shown once!)
	resp := model.WebhookEndpointCreateResponse{
		WebhookEndpointResponse: endpoint.ToResponse(),
		Secret:                  secret,
	}

	writeJSON(w, http.StatusCreated, resp)
}

// List handles GET /api/v1/webhooks
func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpoints, err := h.repo.ListEndpointsByUser(ctx, auth.UserID)
	if err != nil {
		h.logger.Error("failed to list endpoints", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to list webhooks",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	resp := make([]model.WebhookEndpointResponse, len(endpoints))
	for i, ep := range endpoints {
		resp[i] = ep.ToResponse()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"webhooks": resp,
	})
}

// Get handles GET /api/v1/webhooks/{id}
func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to get webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	// Check ownership
	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	writeJSON(w, http.StatusOK, endpoint.ToResponse())
}

// Update handles PATCH /api/v1/webhooks/{id}
func (h *WebhookHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to update webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	var req model.WebhookEndpointUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
			Error: "Invalid request body",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		endpoint.Name = *req.Name
	}
	if req.Description != nil {
		endpoint.Description = *req.Description
	}
	if req.TargetURL != nil {
		if err := webhook.ValidateTargetURLWithOptions(*req.TargetURL, webhook.ValidationOptions{AllowInsecure: h.allowInsecure}); err != nil {
			writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
				Error: err.Error(),
				Code:  "INVALID_URL",
			})
			return
		}
		endpoint.TargetURL = *req.TargetURL
	}
	if req.Enabled != nil {
		endpoint.Enabled = *req.Enabled
	}
	if req.EventTypes != nil {
		for _, et := range *req.EventTypes {
			if !model.IsValidEventType(et) {
				writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{
					Error: "Invalid event type: " + string(et),
					Code:  "INVALID_EVENT_TYPE",
				})
				return
			}
		}
		endpoint.EventTypes = *req.EventTypes
	}

	if err := h.repo.UpdateEndpoint(ctx, endpoint); err != nil {
		h.logger.Error("failed to update endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to update webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	h.logger.Info("webhook endpoint updated",
		"endpoint_id", endpoint.ID,
		"user_id", auth.UserID,
	)

	writeJSON(w, http.StatusOK, endpoint.ToResponse())
}

// Delete handles DELETE /api/v1/webhooks/{id}
func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to delete webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	if err := h.repo.DeleteEndpoint(ctx, endpointID); err != nil {
		h.logger.Error("failed to delete endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to delete webhook",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	h.logger.Info("webhook endpoint deleted",
		"endpoint_id", endpointID,
		"user_id", auth.UserID,
	)

	w.WriteHeader(http.StatusNoContent)
}

// RotateSecret handles POST /api/v1/webhooks/{id}/rotate-secret
func (h *WebhookHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to rotate secret",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Generate new secret
	newSecret, err := webhook.GenerateSecret()
	if err != nil {
		h.logger.Error("failed to generate secret", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to rotate secret",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if err := h.repo.UpdateEndpointSecret(ctx, endpointID, webhook.HashSecret(newSecret)); err != nil {
		h.logger.Error("failed to update secret", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to rotate secret",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	h.logger.Info("webhook secret rotated",
		"endpoint_id", endpointID,
		"user_id", auth.UserID,
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"secret": newSecret,
	})
}

// ListDeliveries handles GET /api/v1/webhooks/{id}/deliveries
func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to list deliveries",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	// Parse query params
	statuses := r.URL.Query()["status"]
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	deliveries, total, err := h.repo.ListDeliveriesByEndpoint(ctx, endpointID, statuses, perPage, offset)
	if err != nil {
		h.logger.Error("failed to list deliveries", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to list deliveries",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	resp := make([]model.WebhookDeliveryResponse, len(deliveries))
	for i, d := range deliveries {
		resp[i] = d.ToResponse()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deliveries": resp,
		"pagination": map[string]interface{}{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// RetryDelivery handles POST /api/v1/webhooks/{id}/deliveries/{delivery_id}/retry
func (h *WebhookHandler) RetryDelivery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth := getAuthContext(ctx)
	if auth == nil {
		writeJSON(w, http.StatusUnauthorized, dto.ErrorResponse{
			Error: "Unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	if !auth.HasScope(model.ScopeWebhook) {
		writeJSON(w, http.StatusForbidden, dto.ErrorResponse{
			Error: "Webhook scope required",
			Code:  "FORBIDDEN",
		})
		return
	}

	endpointID := chi.URLParam(r, "id")
	deliveryID := chi.URLParam(r, "deliveryId")

	endpoint, err := h.repo.GetEndpoint(ctx, endpointID)
	if err != nil {
		if errors.Is(err, webhook.ErrEndpointNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Webhook not found",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to get endpoint", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retry delivery",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if endpoint.UserID != auth.UserID {
		writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
			Error: "Webhook not found",
			Code:  "NOT_FOUND",
		})
		return
	}

	if err := h.repo.ResetDeliveryForRetry(ctx, deliveryID); err != nil {
		if errors.Is(err, webhook.ErrDeliveryNotFound) {
			writeJSON(w, http.StatusNotFound, dto.ErrorResponse{
				Error: "Delivery not found or not exhausted",
				Code:  "NOT_FOUND",
			})
			return
		}
		h.logger.Error("failed to retry delivery", "error", err)
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{
			Error: "Failed to retry delivery",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	h.logger.Info("webhook delivery retry requested",
		"delivery_id", deliveryID,
		"endpoint_id", endpointID,
		"user_id", auth.UserID,
	)

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "retry_scheduled",
	})
}

// generateID generates a unique ID (simplified ULID-like).
func generateID() string {
	timestamp := time.Now().UnixNano()
	return strconv.FormatInt(timestamp, 16)
}
