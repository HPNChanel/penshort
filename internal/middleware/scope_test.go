package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/penshort/penshort/internal/auth"
	"github.com/penshort/penshort/internal/model"
)

func TestRequireScope_Authorized(t *testing.T) {
	testCases := []struct {
		name          string
		scopes        []string
		requiredScope string
		wantStatus    int
	}{
		{
			name:          "read scope allows read",
			scopes:        []string{model.ScopeRead},
			requiredScope: model.ScopeRead,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "write scope allows write",
			scopes:        []string{model.ScopeWrite},
			requiredScope: model.ScopeWrite,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "admin allows read",
			scopes:        []string{model.ScopeAdmin},
			requiredScope: model.ScopeRead,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "admin allows write",
			scopes:        []string{model.ScopeAdmin},
			requiredScope: model.ScopeWrite,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "admin allows admin",
			scopes:        []string{model.ScopeAdmin},
			requiredScope: model.ScopeAdmin,
			wantStatus:    http.StatusOK,
		},
		{
			name:          "multiple scopes work",
			scopes:        []string{model.ScopeRead, model.ScopeWrite},
			requiredScope: model.ScopeWrite,
			wantStatus:    http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create auth context
			authCtx := &model.AuthContext{
				KeyID:     "key123",
				KeyPrefix: "abc123",
				UserID:    "user123",
				Scopes:    tc.scopes,
			}

			// Create handler that returns 200
			handler := RequireScope(tc.requiredScope)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			// Create request with auth context
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := auth.ContextWithAuth(req.Context(), authCtx)
			req = req.WithContext(ctx)

			// Record response
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestRequireScope_Forbidden(t *testing.T) {
	testCases := []struct {
		name          string
		scopes        []string
		requiredScope string
	}{
		{
			name:          "read cannot access write",
			scopes:        []string{model.ScopeRead},
			requiredScope: model.ScopeWrite,
		},
		{
			name:          "read cannot access admin",
			scopes:        []string{model.ScopeRead},
			requiredScope: model.ScopeAdmin,
		},
		{
			name:          "write cannot access admin",
			scopes:        []string{model.ScopeWrite},
			requiredScope: model.ScopeAdmin,
		},
		{
			name:          "webhook cannot access write",
			scopes:        []string{model.ScopeWebhook},
			requiredScope: model.ScopeWrite,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authCtx := &model.AuthContext{
				KeyID:     "key123",
				KeyPrefix: "abc123",
				UserID:    "user123",
				Scopes:    tc.scopes,
			}

			handler := RequireScope(tc.requiredScope)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := auth.ContextWithAuth(req.Context(), authCtx)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
			}
		})
	}
}

func TestRequireScope_NoAuthContext(t *testing.T) {
	handler := RequireScope(model.ScopeRead)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestConvenienceMiddleware(t *testing.T) {
	authCtx := &model.AuthContext{
		KeyID:     "key123",
		KeyPrefix: "abc123",
		UserID:    "user123",
		Scopes:    []string{model.ScopeAdmin},
	}

	testCases := []struct {
		name       string
		middleware func() func(http.Handler) http.Handler
	}{
		{"RequireRead", RequireRead},
		{"RequireWrite", RequireWrite},
		{"RequireAdmin", RequireAdmin},
		{"RequireWebhook", RequireWebhook},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := tc.middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := auth.ContextWithAuth(req.Context(), authCtx)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			// Admin should pass all
			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
		})
	}
}
