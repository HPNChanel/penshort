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

func TestRedirect_CacheMissThenHit(t *testing.T) {
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

func TestRedirect_ExpiredLink(t *testing.T) {
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

func newRedirectTestEnv(t *testing.T) (context.Context, *repository.Repository, *cache.Cache, *metrics.InMemoryRecorder, *service.LinkService, *chi.Mux) {
	t.Helper()

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
	redirectHandler := NewRedirectHandler(svc, logger)

	router := chi.NewRouter()
	router.Get("/{shortCode}", redirectHandler.Redirect)

	return ctx, repo, cacheClient, recorder, svc, router
}
