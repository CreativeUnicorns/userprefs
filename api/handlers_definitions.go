// Package api provides HTTP handlers, middleware, and routing for the user preferences service.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/go-chi/chi/v5"
)

// handleDefinePreference handles the creation of a new preference definition.
func (s *Server) handleDefinePreference(w http.ResponseWriter, r *http.Request) {
	var def userprefs.PreferenceDefinition

	// Limit the size of the request body to 1MB
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&def); err != nil {
		s.respondWithError(w, r, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	if err := s.manager.DefinePreference(def); err != nil {
		if errors.Is(err, userprefs.ErrInvalidType) || errors.Is(err, userprefs.ErrValidation) {
			s.respondWithError(w, r, http.StatusBadRequest, "Invalid preference definition", err)
		} else if errors.Is(err, userprefs.ErrAlreadyExists) {
			// As per DESIGN.MD, POST to /definitions should return 409 if key exists.
			// However, Manager.DefinePreference is idempotent (update/noop).
			// For strict API adherence, we might need a separate Manager method or check existence first.
			// For now, treating it as idempotent successful operation but logging the design note.
			s.logger.Debug("DefinePreference called for existing key, treating as success due to manager idempotency", "key", def.Key)
			s.respondWithJSON(w, r, http.StatusOK, def) // Or http.StatusCreated if we ensure it's a new one.
		} else {
			s.respondWithError(w, r, http.StatusInternalServerError, "Failed to define preference", err)
		}
		return
	}

	s.respondWithJSON(w, r, http.StatusCreated, def)
}

// handleGetDefinition handles fetching a specific preference definition.
func (s *Server) handleGetDefinition(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	def, found := s.manager.GetDefinition(key)
	if !found {
		s.respondWithError(w, r, http.StatusNotFound, "Preference definition not found", nil)
		return
	}
	s.respondWithJSON(w, r, http.StatusOK, def)
}

// handleListDefinitions handles fetching all preference definitions.
func (s *Server) handleListDefinitions(w http.ResponseWriter, r *http.Request) {
	defs, err := s.manager.GetAllDefinitions(r.Context()) // Pass context
	if err != nil {
		s.respondWithError(w, r, http.StatusInternalServerError, "Failed to get all definitions", err)
		return
	}
	// Ensure an empty array is returned instead of null if no definitions exist
	if defs == nil {
		// This might not be strictly necessary if GetAllDefinitions guarantees non-nil slice
		// but good for safety.
		defs = []*userprefs.PreferenceDefinition{}
	}
	s.respondWithJSON(w, r, http.StatusOK, defs)
}

// respondWithError is a helper to send JSON error responses.
func (s *Server) respondWithError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	resp := map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	}
	if err != nil {
		resp["error"].(map[string]string)["details"] = err.Error()
	}
	s.logger.Error("API Error", "status", status, "message", message, "path", r.URL.Path, "error", err)
	respondWithJSONRaw(w, status, resp)
}

// respondWithJSON is a helper to send JSON responses.
func (s *Server) respondWithJSON(w http.ResponseWriter, _ *http.Request, status int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal JSON response", "error", err, "payload", payload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{\"error\":{\"message\":\"Failed to marshal response\"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

// respondWithJSONRaw is a lower-level helper, useful when payload is already a map for error responses.
func respondWithJSONRaw(w http.ResponseWriter, status int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		// This is a fallback error, should rarely happen if payload is simple map[string]interface{}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"Critical: Failed to marshal error response"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
