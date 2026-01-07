// Package main is the entrypoint for the Penshort API server.
package main

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/config"
	"github.com/penshort/penshort/internal/handler"
	"github.com/penshort/penshort/internal/metrics"
	"github.com/penshort/penshort/internal/middleware"
	"github.com/penshort/penshort/internal/repository"
	"github.com/penshort/penshort/internal/server"
	"github.com/penshort/penshort/internal/service"
)

func main() {
	// Initialize context
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := initLogger(cfg)

	// Initialize database
	repo, err := repository.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error(
			"failed to connect to database",
			slog.String("error", sanitizeError(err, cfg.DatabaseURL)),
			slog.String("database_url", redactURL(cfg.DatabaseURL)),
		)
		os.Exit(1)
	}
	defer repo.Close()
	logger.Info("connected to database")

	// Initialize cache
	cacheClient, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error(
			"failed to connect to Redis",
			slog.String("error", sanitizeError(err, cfg.RedisURL)),
			slog.String("redis_url", redactURL(cfg.RedisURL)),
		)
		os.Exit(1)
	}
	defer cacheClient.Close()
	logger.Info("connected to Redis")

	// Initialize services
	metricsRecorder := metrics.NewNoop()
	linkService := service.NewLinkService(repo, cacheClient, cfg.BaseURL, metricsRecorder)

	// Initialize handlers
	h := handler.New()
	healthHandler := handler.NewHealthHandler(repo, cacheClient)
	linkHandler := handler.NewLinkHandler(linkService, logger)
	redirectHandler := handler.NewRedirectHandler(linkService, logger)
	apiKeyHandler := handler.NewAPIKeyHandler(logger, repo)

	// Setup router
	r := setupRouter(h, healthHandler, linkHandler, redirectHandler, apiKeyHandler, repo, cacheClient, cfg, logger)

	// Create and run server
	srv := server.New(
		r,
		cfg.AppPort,
		cfg.ReadTimeout,
		cfg.WriteTimeout,
		cfg.ShutdownTimeout,
		logger,
	)

	logger.Info("starting server",
		"port", cfg.AppPort,
		"base_url", cfg.BaseURL,
		"env", cfg.AppEnv,
	)

	if err := srv.Run(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// initLogger initializes the slog logger based on configuration.
func initLogger(cfg *config.Config) *slog.Logger {
	var h slog.Handler

	level := parseLogLevel(cfg.LogLevel)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(h)
	slog.SetDefault(logger)

	return logger
}

// parseLogLevel converts string log level to slog.Level.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// setupRouter configures the chi router with all routes and middleware.
func setupRouter(
	h *handler.Handler,
	healthHandler *handler.HealthHandler,
	linkHandler *handler.LinkHandler,
	redirectHandler *handler.RedirectHandler,
	apiKeyHandler *handler.APIKeyHandler,
	repo *repository.Repository,
	cacheClient *cache.Cache,
	cfg *config.Config,
	logger *slog.Logger,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recoverer(logger))

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	// Root info endpoint
	r.Get("/", h.Hello)

	// Auth middleware configuration
	authCfg := middleware.AuthConfig{
		Logger:     logger,
		Repository: repo,
		Cache:      cacheClient,
	}

	// Rate limit middleware configuration
	rateLimitCfg := middleware.RateLimitConfig{
		Logger:           logger,
		Cache:            cacheClient,
		APIEnabled:       cfg.RateLimitAPIEnabled,
		RedirectEnabled:  cfg.RateLimitRedirectEnabled,
		RedirectRPS:      cfg.RateLimitRedirectRPS,
		RedirectBurst:    cfg.RateLimitRedirectBurst,
	}

	// API v1 routes (require authentication)
	r.Route("/api/v1", func(r chi.Router) {
		// Apply auth and rate limit middleware to all API routes
		r.Use(middleware.Auth(authCfg))
		r.Use(middleware.RateLimitAPI(rateLimitCfg))

		// Link management (requires write scope for mutations)
		r.Route("/links", func(r chi.Router) {
			r.With(middleware.RequireRead()).Get("/", linkHandler.List)
			r.With(middleware.RequireRead()).Get("/{id}", linkHandler.Get)
			r.With(middleware.RequireWrite()).Post("/", linkHandler.Create)
			r.With(middleware.RequireWrite()).Patch("/{id}", linkHandler.Update)
			r.With(middleware.RequireAdmin()).Delete("/{id}", linkHandler.Delete)
		})

		// API key management (requires admin scope for mutations)
		r.Route("/api-keys", func(r chi.Router) {
			r.With(middleware.RequireRead()).Get("/", apiKeyHandler.ListAPIKeys)
			r.With(middleware.RequireAdmin()).Post("/", apiKeyHandler.CreateAPIKey)
			r.With(middleware.RequireAdmin()).Delete("/{key_id}", apiKeyHandler.RevokeAPIKey)
			r.With(middleware.RequireAdmin()).Post("/{key_id}/rotate", apiKeyHandler.RotateAPIKey)
		})
	})

	// Redirect handler with IP-based rate limiting (no auth required)
	r.With(middleware.RateLimitIP(rateLimitCfg)).Get("/{shortCode}", redirectHandler.Redirect)

	// 404 and 405 handlers
	r.NotFound(h.NotFound)
	r.MethodNotAllowed(h.MethodNotAllowed)

	return r
}

var passwordPattern = regexp.MustCompile(`(?i)password=[^\s]+`)

func redactURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "[redacted]"
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		if username == "" {
			parsed.User = url.User("redacted")
		} else {
			parsed.User = url.User(username)
		}
	}

	return parsed.String()
}

func sanitizeError(err error, secrets ...string) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		redacted := redactURL(secret)
		if redacted == "" {
			redacted = "[redacted]"
		}
		msg = strings.ReplaceAll(msg, secret, redacted)
	}

	return passwordPattern.ReplaceAllString(msg, "password=redacted")
}
