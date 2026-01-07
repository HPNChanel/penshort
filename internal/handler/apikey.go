package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
	"github.com/oklog/ulid/v2"
)

// APIKeyHandler handles API key management endpoints.
type APIKeyHandler struct {
	logger     *slog.Logger
	repository *repository.Repository
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(logger *slog.Logger, repo *repository.Repository) *APIKeyHandler {
	return &APIKeyHandler{
		logger:     logger,
		repository: repo,
	}
}

// CreateAPIKey handles POST /v1/api-keys
func (h *APIKeyHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.AuthFromContext(ctx)
	if authCtx == nil {
		writeAPIKeyError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Parse request body
	var req model.APIKeyCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIKeyError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate scopes
	for _, scope := range req.Scopes {
		if !slices.Contains(model.ValidScopes, scope) {
			writeAPIKeyError(w, http.StatusBadRequest, "INVALID_SCOPE",
				"Invalid scope: "+scope+". Valid scopes: read, write, webhook, admin")
			return
		}
	}

	// Default to read scope if none provided
	if len(req.Scopes) == 0 {
		req.Scopes = []string{model.ScopeRead}
	}

	// Generate new key
	generatedKey, err := auth.GenerateAPIKey(auth.EnvLive)
	if err != nil {
		h.logger.Error("failed to generate API key", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate API key")
		return
	}

	// Create API key entity
	apiKey := &model.APIKey{
		ID:            ulid.Make().String(),
		UserID:        authCtx.UserID,
		KeyHash:       generatedKey.Hash,
		KeyPrefix:     generatedKey.Prefix,
		Scopes:        req.Scopes,
		RateLimitTier: model.TierFree,
		Name:          req.Name,
		CreatedAt:     time.Now(),
	}

	// Store in database
	if err := h.repository.CreateAPIKey(ctx, apiKey); err != nil {
		h.logger.Error("failed to create API key", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create API key")
		return
	}

	h.logger.Info("API key created",
		slog.String("key_id", apiKey.ID),
		slog.String("key_prefix", apiKey.KeyPrefix),
		slog.String("user_id", apiKey.UserID),
	)

	// Return response with plaintext key (shown once only!)
	response := model.APIKeyCreateResponse{
		ID:            apiKey.ID,
		Key:           generatedKey.Plaintext,
		Name:          apiKey.Name,
		KeyPrefix:     apiKey.KeyPrefix,
		Scopes:        apiKey.Scopes,
		RateLimitTier: apiKey.RateLimitTier,
		CreatedAt:     apiKey.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// ListAPIKeys handles GET /v1/api-keys
func (h *APIKeyHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.AuthFromContext(ctx)
	if authCtx == nil {
		writeAPIKeyError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	keys, err := h.repository.ListAPIKeysByUserID(ctx, authCtx.UserID)
	if err != nil {
		h.logger.Error("failed to list API keys", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list API keys")
		return
	}

	// Convert to response format (without secrets)
	responses := make([]model.APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		responses = append(responses, key.ToResponse())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": responses})
}

// RevokeAPIKey handles DELETE /v1/api-keys/{key_id}
func (h *APIKeyHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.AuthFromContext(ctx)
	if authCtx == nil {
		writeAPIKeyError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Extract key_id from path
	keyID := r.PathValue("key_id")
	if keyID == "" {
		writeAPIKeyError(w, http.StatusBadRequest, "INVALID_REQUEST", "Key ID is required")
		return
	}

	// Verify key belongs to user
	key, err := h.repository.GetAPIKeyByID(ctx, keyID)
	if err != nil {
		// Return 404 for both not found and already revoked (security)
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found or already revoked")
		return
	}

	if key.UserID != authCtx.UserID {
		// Return 404 to prevent enumeration
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found or already revoked")
		return
	}

	if key.IsRevoked() {
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found or already revoked")
		return
	}

	// Revoke the key
	if err := h.repository.RevokeAPIKey(ctx, keyID); err != nil {
		h.logger.Error("failed to revoke API key", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke API key")
		return
	}

	h.logger.Info("API key revoked",
		slog.String("key_id", keyID),
		slog.String("user_id", authCtx.UserID),
	)

	w.WriteHeader(http.StatusNoContent)
}

// RotateAPIKey handles POST /v1/api-keys/{key_id}/rotate
func (h *APIKeyHandler) RotateAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := auth.AuthFromContext(ctx)
	if authCtx == nil {
		writeAPIKeyError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}

	// Extract key_id from path
	keyID := r.PathValue("key_id")
	if keyID == "" {
		writeAPIKeyError(w, http.StatusBadRequest, "INVALID_REQUEST", "Key ID is required")
		return
	}

	// Get the existing key
	oldKey, err := h.repository.GetAPIKeyByID(ctx, keyID)
	if err != nil {
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found")
		return
	}

	if oldKey.UserID != authCtx.UserID {
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found")
		return
	}

	if oldKey.IsRevoked() {
		writeAPIKeyError(w, http.StatusNotFound, "KEY_NOT_FOUND", "API key not found or already revoked")
		return
	}

	// Generate new key with same properties
	generatedKey, err := auth.GenerateAPIKey(auth.EnvLive)
	if err != nil {
		h.logger.Error("failed to generate API key", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate API key")
		return
	}

	now := time.Now()

	// Create new API key entity
	newKey := &model.APIKey{
		ID:            ulid.Make().String(),
		UserID:        oldKey.UserID,
		KeyHash:       generatedKey.Hash,
		KeyPrefix:     generatedKey.Prefix,
		Scopes:        oldKey.Scopes,
		RateLimitTier: oldKey.RateLimitTier,
		Name:          oldKey.Name,
		CreatedAt:     now,
	}

	// Create new key first
	if err := h.repository.CreateAPIKey(ctx, newKey); err != nil {
		h.logger.Error("failed to create rotated API key", slog.String("error", err.Error()))
		writeAPIKeyError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rotate API key")
		return
	}

	// Revoke old key
	if err := h.repository.RevokeAPIKey(ctx, oldKey.ID); err != nil {
		h.logger.Error("failed to revoke old API key during rotation", slog.String("error", err.Error()))
		// Continue - new key is already created
	}

	h.logger.Info("API key rotated",
		slog.String("old_key_id", oldKey.ID),
		slog.String("new_key_id", newKey.ID),
		slog.String("user_id", authCtx.UserID),
	)

	// Return response
	response := model.APIKeyRotateResponse{
		OldKeyID:        oldKey.ID,
		OldKeyRevokedAt: now,
		NewKey: model.APIKeyCreateResponse{
			ID:            newKey.ID,
			Key:           generatedKey.Plaintext,
			Name:          newKey.Name,
			KeyPrefix:     newKey.KeyPrefix,
			Scopes:        newKey.Scopes,
			RateLimitTier: newKey.RateLimitTier,
			CreatedAt:     newKey.CreatedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// writeAPIKeyError writes a JSON error response.
func writeAPIKeyError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
