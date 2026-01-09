package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/penshort/penshort/internal/analytics"
	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/model"
	"github.com/penshort/penshort/internal/repository"
	"github.com/penshort/penshort/internal/service"
	"github.com/penshort/penshort/internal/testutil"
)

func TestAnalyticsIngestAndQuery(t *testing.T) {
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
		t.Fatalf("reset links schema: %v", err)
	}
	if err := testutil.ResetAnalyticsSchema(ctx, repo.Pool()); err != nil {
		t.Fatalf("reset analytics schema: %v", err)
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
	linkService := service.NewLinkService(repo, cacheClient, "http://localhost:8080", recorder)
	clickRepo := repository.NewClickEventRepository(repo)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	publisher := analytics.NewPublisher(cacheClient.Client(), logger, recorder)
	redirectHandler := NewRedirectHandler(linkService, publisher, logger)
	analyticsHandler := NewAnalyticsHandler(clickRepo, logger)

	worker := analytics.NewWorker(cacheClient.Client(), clickRepo, logger, "test-consumer", recorder)
	worker.SetBlockTimeout(200 * time.Millisecond)
	worker.SetClaimInterval(200 * time.Millisecond)
	worker.SetMetricsInterval(200 * time.Millisecond)
	worker.SetBatchSize(100)

	workerCtx, cancel := context.WithCancel(ctx)
	workerErr := make(chan error, 1)
	go func() {
		workerErr <- worker.Run(workerCtx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-workerErr:
		case <-time.After(2 * time.Second):
		}
	})

	alias := fmt.Sprintf("analytics-%d", time.Now().UnixNano())
	link, err := linkService.CreateLink(ctx, service.CreateLinkInput{
		Destination: "https://example.com/analytics",
		Alias:       alias,
	})
	if err != nil {
		t.Fatalf("create link: %v", err)
	}

	router := chi.NewRouter()
	router.Get("/{shortCode}", redirectHandler.Redirect)
	router.Get("/api/v1/links/{id}/analytics", analyticsHandler.GetLinkAnalytics)

	sendRedirect(t, router, alias, "203.0.113.10", "TestAgent/1.0")
	sendRedirect(t, router, alias, "203.0.113.10", "TestAgent/1.0")
	sendRedirect(t, router, alias, "203.0.113.11", "TestAgent/1.0")

	date := time.Now().UTC().Format("2006-01-02")
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		response, status := fetchAnalytics(t, router, link.ID, date, date)
		if status != http.StatusOK {
			t.Fatalf("analytics status %d", status)
		}
		if response.Summary.TotalClicks == 3 && response.Summary.UniqueVisitors == 2 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	response, _ := fetchAnalytics(t, router, link.ID, date, date)
	t.Fatalf("expected totals 3/2, got %d/%d", response.Summary.TotalClicks, response.Summary.UniqueVisitors)
}

func sendRedirect(t *testing.T, router *chi.Mux, alias, ip, ua string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/"+alias, nil)
	req.Header.Set("CF-Connecting-IP", ip)
	req.Header.Set("User-Agent", ua)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound && rec.Code != http.StatusMovedPermanently {
		t.Fatalf("unexpected redirect status %d", rec.Code)
	}
}

func fetchAnalytics(t *testing.T, router *chi.Mux, linkID, from, to string) (model.AnalyticsResponse, int) {
	t.Helper()

	path := fmt.Sprintf("/api/v1/links/%s/analytics?from=%s&to=%s", linkID, from, to)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload model.AnalyticsResponse
	if rec.Code == http.StatusOK {
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decode analytics response: %v", err)
		}
	}

	return payload, rec.Code
}
