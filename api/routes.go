package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes() {
	// Middleware stack
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(LoggerMiddleware(s.logger)) // Custom logger middleware
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.SetHeader("Content-Type", "application/json"))

	// API versioning group
	s.router.Route("/api/v1", func(r chi.Router) {
		// Health check endpoint
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK")) // Best effort write
		})

		// Preference Definitions Endpoints
		r.Route("/definitions", func(r chi.Router) {
			r.Post("/", s.handleDefinePreference)  // POST /api/v1/definitions
			r.Get("/{key}", s.handleGetDefinition) // GET /api/v1/definitions/{key}
			r.Get("/", s.handleListDefinitions)    // GET /api/v1/definitions
			// r.Put("/{key}", s.handleUpdateDefinition)    // PUT /api/v1/definitions/{key} (To be implemented)
			// r.Delete("/{key}", s.handleDeleteDefinition)  // DELETE /api/v1/definitions/{key} (To be implemented)
		})

		// User Preferences Endpoints (To be implemented)
		// r.Route("/users/{userID}/preferences", func(r chi.Router) {
		// 	r.Get("/{key}", s.handleGetUserPreference)
		// 	r.Put("/{key}", s.handleSetUserPreference)
		// 	r.Delete("/{key}", s.handleDeleteUserPreference)
		// 	r.Get("/", s.handleGetAllUserPreferences)
		// 	r.Delete("/", s.handleDeleteAllUserPreferences)
		// })
	})
}
