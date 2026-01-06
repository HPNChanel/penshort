// Package server provides HTTP server lifecycle management.
// Includes graceful shutdown handling for production deployments.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps http.Server with graceful shutdown.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
}

// New creates a new Server instance.
func New(handler http.Handler, port int, readTimeout, writeTimeout, shutdownTimeout time.Duration, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		shutdownTimeout: shutdownTimeout,
		logger:          logger,
	}
}

// Run starts the server and blocks until shutdown signal is received.
// It handles graceful shutdown on SIGINT/SIGTERM.
func (s *Server) Run() error {
	// Channel to receive shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Channel to receive server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		s.logger.Info("server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		s.logger.Info("shutdown signal received", "signal", sig.String())
		return s.gracefulShutdown()
	}
}

// gracefulShutdown attempts to gracefully shut down the server.
func (s *Server) gracefulShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	s.logger.Info("shutting down server", "timeout", s.shutdownTimeout)

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("server stopped gracefully")
	return nil
}
