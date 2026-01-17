package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/handler/dto"
	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
	"github.com/penshort/penshort/internal/service"
	"github.com/penshort/penshort/internal/testutil"
)

func TestIntegrationRedirect_CacheMissThenHit(t *testing.T) {
	ctx, _, cacheClient, recorder, svc, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("cache-%d", time.Now().UnixNano())
	destination := "https://example.com/cache"

	link, err := svc.CreateLink(ctx, service.CreateLinkInput{
		Destination: destination,
		Alias:       alias,
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != int(link.RedirectType) {
		t.Fatalf("expected status %d, got %d", link.RedirectType, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != destination {
		t.Fatalf("expected Location %q, got %q", destination, location)
	}

	snap := recorder.Snapshot()
	if snap.RedirectCacheMisses != 1 || snap.RedirectCacheHits != 0 {
		t.Fatalf("unexpected cache counters: hits=%d misses=%d", snap.RedirectCacheHits, snap.RedirectCacheMisses)
	}

	if _, err := cacheClient.GetLink(ctx, alias); err != nil {
		t.Fatalf("expected cached link, got %v", err)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != int(link.RedirectType) {
		t.Fatalf("expected status %d, got %d", link.RedirectType, rec2.Code)
	}
	if location := rec2.Header().Get("Location"); location != destination {
		t.Fatalf("expected Location %q, got %q", destination, location)
	}

	snap2 := recorder.Snapshot()
	if snap2.RedirectCacheHits != 1 || snap2.RedirectCacheMisses != 1 {
		t.Fatalf("unexpected cache counters after hit: hits=%d misses=%d", snap2.RedirectCacheHits, snap2.RedirectCacheMisses)
	}
}

func TestIntegrationRedirect_ExpiredLink(t *testing.T) {
	ctx, repo, cacheClient, _, _, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("expired-%d", time.Now().UnixNano())
	now := time.Now().UTC()
	expiredAt := now.Add(-1 * time.Minute)

	link := &model.Link{
		ID:           fmt.Sprintf("expired-%d", now.UnixNano()),
		ShortCode:    alias,
		Destination:  "https://example.com/expired",
		RedirectType: model.RedirectTemporary,
		OwnerID:      "system",
		Enabled:      true,
		ExpiresAt:    &expiredAt,
		ClickCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create expired link: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("expected status %d, got %d", http.StatusGone, rec.Code)
	}

	var payload dto.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "LINK_EXPIRED" {
		t.Fatalf("expected LINK_EXPIRED, got %q", payload.Code)
	}

	if _, err := cacheClient.GetLink(ctx, alias); !errors.Is(err, cache.ErrCacheMiss) {
		t.Fatalf("expected cache miss for expired link, got %v", err)
	}
}

func TestIntegrationRedirect_CacheInvalidationOnUpdate(t *testing.T) {
	ctx, repo, cacheClient, _, svc, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("invalidate-%d", time.Now().UnixNano())
	destination := "https://example.com/original"

	link, err := svc.CreateLink(ctx, service.CreateLinkInput{
		Destination: destination,
		Alias:       alias,
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	// First request - cache miss, populates cache
	req1 := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != int(link.RedirectType) {
		t.Fatalf("expected status %d, got %d", link.RedirectType, rec1.Code)
	}

	// Verify cache is populated
	cachedLink, err := cacheClient.GetLink(ctx, alias)
	if err != nil {
		t.Fatalf("expected cached link, got %v", err)
	}
	if cachedLink.Destination != destination {
		t.Fatalf("cached destination mismatch: got %q, want %q", cachedLink.Destination, destination)
	}

	// Update link destination directly in DB
	newDestination := "https://example.com/updated"
	link.Destination = newDestination
	if err := repo.UpdateLink(ctx, link); err != nil {
		t.Fatalf("update link: %v", err)
	}

	// Invalidate cache
	if err := cacheClient.InvalidateLink(ctx, alias); err != nil {
		t.Fatalf("invalidate cache: %v", err)
	}

	// Next request should hit DB and get new destination
	req2 := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Header().Get("Location") != newDestination {
		t.Errorf("expected Location %q after invalidation, got %q", newDestination, rec2.Header().Get("Location"))
	}
}

func TestIntegrationRedirect_ExpiryBoundaryTime(t *testing.T) {
	ctx, repo, _, _, _, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("boundary-%d", time.Now().UnixNano())
	now := time.Now().UTC()

	// Link that expires in 500ms
	expiryTime := now.Add(500 * time.Millisecond)
	link := &model.Link{
		ID:           fmt.Sprintf("boundary-%d", now.UnixNano()),
		ShortCode:    alias,
		Destination:  "https://example.com/boundary",
		RedirectType: model.RedirectTemporary,
		OwnerID:      "system",
		Enabled:      true,
		ExpiresAt:    &expiryTime,
		ClickCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link: %v", err)
	}

	// Request before expiry - should redirect
	req1 := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusFound {
		t.Errorf("before expiry: expected 302, got %d", rec1.Code)
	}

	// Wait for expiry
	time.Sleep(600 * time.Millisecond)

	// Request after expiry - should return 410 Gone
	req2 := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusGone {
		t.Errorf("after expiry: expected 410, got %d", rec2.Code)
	}
}

func TestIntegrationRedirect_DisabledLink(t *testing.T) {
	ctx, repo, _, _, _, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("disabled-%d", time.Now().UnixNano())
	now := time.Now().UTC()

	link := &model.Link{
		ID:           fmt.Sprintf("disabled-%d", now.UnixNano()),
		ShortCode:    alias,
		Destination:  "https://example.com/disabled",
		RedirectType: model.RedirectTemporary,
		OwnerID:      "system",
		Enabled:      false, // Disabled
		ClickCount:   0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.CreateLink(ctx, link); err != nil {
		t.Fatalf("create link: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// Disabled links should return appropriate error
	if rec.Code == http.StatusFound || rec.Code == http.StatusMovedPermanently {
		t.Errorf("disabled link should not redirect, got status %d", rec.Code)
	}
}

func TestIntegrationRedirect_NullExpiry(t *testing.T) {
	ctx, _, _, _, svc, router := newRedirectTestEnv(t)

	alias := fmt.Sprintf("noexpiry-%d", time.Now().UnixNano())

	// Create link without expiry
	link, err := svc.CreateLink(ctx, service.CreateLinkInput{
		Destination: "https://example.com/noexpiry",
		Alias:       alias,
		// No ExpiresAt
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	// Should always redirect
	req := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != int(link.RedirectType) {
		t.Errorf("expected status %d, got %d", link.RedirectType, rec.Code)
	}
	if rec.Header().Get("Location") != "https://example.com/noexpiry" {
		t.Errorf("unexpected Location: %q", rec.Header().Get("Location"))
	}
}

func newRedirectTestEnv(t *testing.T) (context.Context, *repository.Repository, *cache.Cache, *metrics.InMemoryRecorder, *service.LinkService, *chi.Mux) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx := context.Background()
	dbURL := testutil.RequireEnv(t, "DATABASE_URL")
	redisURL := testutil.RequireEnv(t, "REDIS_URL")

	repo, err := repository.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(repo.Close)

	unlock, err := testutil.AcquireDBLock(ctx, repo.Pool())
	if err != nil {
		t.Fatalf("acquire db lock: %v", err)
	}
	t.Cleanup(func() {
		_ = unlock()
	})

	if err := testutil.ResetLinksSchema(ctx, repo.Pool()); err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	cacheClient, err := cache.New(ctx, redisURL)
	if err != nil {
		t.Fatalf("connect redis: %v", err)
	}
	t.Cleanup(func() {
		_ = cacheClient.Close()
	})

	if err := testutil.FlushRedis(ctx, cacheClient.Client()); err != nil {
		t.Fatalf("flush redis: %v", err)
	}

	recorder := metrics.NewInMemory()
	svc := service.NewLinkService(repo, cacheClient, "http://localhost:8080", recorder)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	redirectHandler := NewRedirectHandler(svc, nil, logger)

	router := chi.NewRouter()
	router.Get("/{shortCode}", redirectHandler.Redirect)

	return ctx, repo, cacheClient, recorder, svc, router
}
