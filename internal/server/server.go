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
	"sync"
	"syscall"
	"time"
)

// ShutdownFunc is a function that shuts down a component gracefully.
type ShutdownFunc func(ctx context.Context) error

// Server wraps http.Server with graceful shutdown.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
	logger          *slog.Logger
	shutdownFuncs   []ShutdownFunc
	mu              sync.Mutex
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
		shutdownFuncs:   make([]ShutdownFunc, 0),
	}
}

// OnShutdown registers a function to be called during graceful shutdown.
// Shutdown functions are called in reverse order (LIFO) after the HTTP server stops.
// This allows workers to be registered first and shut down last.
func (s *Server) OnShutdown(name string, fn ShutdownFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownFuncs = append(s.shutdownFuncs, func(ctx context.Context) error {
		s.logger.Info("shutting down component", "name", name)
		if err := fn(ctx); err != nil {
			s.logger.Error("component shutdown error", "name", name, "error", err)
			return err
		}
		s.logger.Info("component stopped", "name", name)
		return nil
	})
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

// gracefulShutdown attempts to gracefully shut down the server and all registered components.
func (s *Server) gracefulShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// Phase 1: Stop accepting new connections
	s.logger.Info("phase 1: stopping HTTP server", "timeout", s.shutdownTimeout)
	s.httpServer.SetKeepAlivesEnabled(false)

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
		// Continue with other shutdowns even if HTTP fails
	}
	s.logger.Info("HTTP server stopped")

	// Phase 2: Shutdown registered components in reverse order
	s.logger.Info("phase 2: stopping registered components", "count", len(s.shutdownFuncs))

	var errs []error
	s.mu.Lock()
	funcs := s.shutdownFuncs
	s.mu.Unlock()

	// Reverse order - last registered shuts down first
	for i := len(funcs) - 1; i >= 0; i-- {
		if err := funcs[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		s.logger.Error("shutdown completed with errors", "error_count", len(errs))
		return errs[0]
	}

	s.logger.Info("server stopped gracefully")
	return nil
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
