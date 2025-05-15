package storage

import (
	"context"
	"sync"
	"time"

	"github.com/CreativeUnicorns/userprefs"
)

// MemoryStorage implements the Storage interface using an in-memory map.
// This is useful for testing or simple applications where persistence is not required.
type MemoryStorage struct {
	mu    sync.RWMutex
	prefs map[string]map[string]*userprefs.Preference // userID -> key -> Preference
}

// NewMemoryStorage creates a new instance of MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		prefs: make(map[string]map[string]*userprefs.Preference),
	}
}

// Get retrieves a preference for a given user ID and key.
// It returns userprefs.ErrNotFound if the preference does not exist.
func (s *MemoryStorage) Get(_ context.Context, userID, key string) (*userprefs.Preference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPrefs, ok := s.prefs[userID]
	if !ok {
		return nil, userprefs.ErrNotFound
	}

	pref, ok := userPrefs[key]
	if !ok {
		return nil, userprefs.ErrNotFound
	}

	// Return a copy to prevent modification of the stored preference through the pointer
	prefCopy := *pref
	return &prefCopy, nil
}

// Set stores a preference.
// It updates the UpdatedAt field to the current time.
func (s *MemoryStorage) Set(_ context.Context, pref *userprefs.Preference) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.prefs[pref.UserID]; !ok {
		s.prefs[pref.UserID] = make(map[string]*userprefs.Preference)
	}

	// Make a copy to store, ensuring original pref is not modified by storage
	// and to manage UpdatedAt consistently.
	prefToStore := *pref
	prefToStore.UpdatedAt = time.Now()
	s.prefs[pref.UserID][pref.Key] = &prefToStore
	return nil
}

// Delete removes a preference for a given user ID and key.
// It returns nil if the preference does not exist or on successful deletion.
func (s *MemoryStorage) Delete(_ context.Context, userID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userPrefs, ok := s.prefs[userID]
	if !ok {
		return nil // Or userprefs.ErrNotFound if strict error desired
	}

	delete(userPrefs, key)
	// If the user has no more preferences, remove the user's map entry
	if len(userPrefs) == 0 {
		delete(s.prefs, userID)
	}
	return nil
}

// GetAll retrieves all preferences for a given user ID.
func (s *MemoryStorage) GetAll(_ context.Context, userID string) (map[string]*userprefs.Preference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPrefs, ok := s.prefs[userID]
	if !ok {
		return make(map[string]*userprefs.Preference), nil // Return empty map if user not found
	}

	// Return copies to prevent modification
	prefsCopy := make(map[string]*userprefs.Preference, len(userPrefs))
	for k, v := range userPrefs {
		valCopy := *v
		prefsCopy[k] = &valCopy
	}
	return prefsCopy, nil
}

// GetByCategory retrieves all preferences for a given user ID and category.
func (s *MemoryStorage) GetByCategory(_ context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userPrefs, ok := s.prefs[userID]
	if !ok {
		return make(map[string]*userprefs.Preference), nil // Return empty map if user not found
	}

	result := make(map[string]*userprefs.Preference)
	for key, pref := range userPrefs {
		if pref.Category == category {
			prefCopy := *pref
			result[key] = &prefCopy
		}
	}
	return result, nil
}

// Close is a no-op for MemoryStorage as there are no external resources to release.
func (s *MemoryStorage) Close() error {
	return nil
}
