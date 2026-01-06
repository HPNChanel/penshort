// Package main is the entrypoint for the Penshort API server.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/penshort/penshort/internal/cache"
	"github.com/penshort/penshort/internal/config"
	"github.com/penshort/penshort/internal/handler"
	"github.com/penshort/penshort/internal/middleware"
	"github.com/penshort/penshort/internal/repository"
	"github.com/penshort/penshort/internal/server"
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
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()
	logger.Info("connected to database")

	// Initialize cache
	cacheClient, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer cacheClient.Close()
	logger.Info("connected to Redis")

	// Initialize handlers
	h := handler.New()
	healthHandler := handler.NewHealthHandler(repo, cacheClient)

	// Setup router
	r := setupRouter(h, healthHandler, logger)

	// Create and run server
	srv := server.New(
		r,
		cfg.AppPort,
		cfg.ReadTimeout,
		cfg.WriteTimeout,
		cfg.ShutdownTimeout,
		logger,
	)

	if err := srv.Run(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

// initLogger initializes the slog logger based on configuration.
func initLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	level := parseLogLevel(cfg.LogLevel)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
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
func setupRouter(h *handler.Handler, healthHandler *handler.HealthHandler, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Recoverer(logger))

	// Health endpoints (no auth required)
	r.Get("/healthz", healthHandler.Healthz)
	r.Get("/readyz", healthHandler.Readyz)

	// API routes
	r.Get("/", h.Hello)

	// 404 and 405 handlers
	r.NotFound(h.NotFound)
	r.MethodNotAllowed(h.MethodNotAllowed)

	return r
}
