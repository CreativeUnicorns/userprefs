package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/go-chi/chi/v5"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	manager    *userprefs.Manager
	logger     userprefs.Logger
	router     *chi.Mux
	httpServer *http.Server
}

// Config holds configuration for the API server.
// TODO: Expand with Address, TLS config, timeouts etc.
type Config struct {
	ListenAddress string
	Manager       *userprefs.Manager
	Logger        userprefs.Logger
}

// NewServer creates and configures a new API server instance.
func NewServer(cfg Config) (*Server, error) {
	if cfg.Manager == nil {
		return nil, fmt.Errorf("manager is required")
	}
	if cfg.Logger == nil {
		// Fallback to a default logger if none provided, though ideally one should always be injected.
		cfg.Logger = userprefs.NewDefaultLogger()
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":8080" // Default listen address
	}

	s := &Server{
		manager: cfg.Manager,
		logger:  cfg.Logger,
		router:  chi.NewRouter(),
	}

	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: s.router,
		// Configure timeouts to prevent resource exhaustion
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// Start runs the HTTP server.
// This method is blocking and will only return when the server is shut down
// or an unrecoverable error occurs (e.g., failure to bind to the address).
// If non-blocking behavior is desired, the caller should run this method
// in a separate goroutine.
// It logs the server startup and shutdown events.
// Returns http.ErrServerClosed if the server is gracefully shut down, nil otherwise for graceful shutdown scenarios handled by ListenAndServe itself,
// or an error if the server fails to start or stops unexpectedly.
func (s *Server) Start() error {
	s.logger.Info("API server starting", "address", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not start server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("API server stopping")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}
	s.logger.Info("API server stopped gracefully")
	return nil
}
