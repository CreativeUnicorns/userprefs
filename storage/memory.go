package storage

import (
	"context"
	"sync"
	"time"

	"github.com/CreativeUnicorns/userprefs"
)

// MemoryStorage implements the userprefs.Storage interface using an in-memory map.
// It is primarily useful for testing or in scenarios where data persistence
// across application restarts is not required.
//
// MemoryStorage is safe for concurrent use by multiple goroutines due to its
// internal use of a sync.RWMutex to synchronize access to the preferences map.
// The internal map `prefs` stores preferences nested by userID and then by preference key.
type MemoryStorage struct {
	mu    sync.RWMutex
	prefs map[string]map[string]*userprefs.Preference // userID -> key -> Preference
}

// NewMemoryStorage creates and returns a new, initialized instance of MemoryStorage.
// The returned MemoryStorage is ready for immediate use.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		prefs: make(map[string]map[string]*userprefs.Preference),
	}
}

// Get retrieves a specific preference for a given user ID and key.
// The provided context.Context is not used by this in-memory implementation but is
// part of the userprefs.Storage interface contract.
//
// If the preference is found, it returns a *copy* of the userprefs.Preference and a nil error.
// Returning a copy ensures that modifications to the retrieved preference do not affect
// the data stored in MemoryStorage.
// If the preference for the given userID and key does not exist, it returns nil and
// userprefs.ErrNotFound.
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

// Set stores or updates a user's preference.
// The provided context.Context is not used by this in-memory implementation but is
// part of the userprefs.Storage interface contract.
//
// A *copy* of the provided userprefs.Preference is stored to prevent external modifications
// from affecting the data within MemoryStorage.
// The UpdatedAt field of the stored preference is automatically set to the current time.
// This method always returns a nil error.
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

// Delete removes a specific preference for a given user ID and key.
// The provided context.Context is not used by this in-memory implementation but is
// part of the userprefs.Storage interface contract.
//
// If the preference for the given userID and key does not exist, it returns
// userprefs.ErrNotFound. Otherwise, it deletes the preference and returns nil.
// If the deletion results in a user having no more preferences, the entry for
// that user is removed from the internal map to save space.
func (s *MemoryStorage) Delete(_ context.Context, userID, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	userPrefs, ok := s.prefs[userID]
	if !ok {
		return userprefs.ErrNotFound
	}

	if _, keyOk := userPrefs[key]; !keyOk {
		return userprefs.ErrNotFound // Key not found within user's prefs
	}

	delete(userPrefs, key)
	// If the user has no more preferences, remove the user's map entry
	if len(userPrefs) == 0 {
		delete(s.prefs, userID)
	}
	return nil
}

// GetAll retrieves all preferences associated with the given user ID.
// The provided context.Context is not used by this in-memory implementation but is
// part of the userprefs.Storage interface contract.
//
// It returns a map where keys are preference keys and values are *copies* of
// userprefs.Preference objects. Returning copies ensures immutability of stored data.
// If the user ID is not found or the user has no preferences, an empty map and a nil error
// are returned.
// This method always returns a nil error.
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

// GetByCategory retrieves all preferences for a given user ID that belong to the specified category.
// The provided context.Context is not used by this in-memory implementation but is
// part of the userprefs.Storage interface contract.
//
// It returns a map where keys are preference keys and values are *copies* of
// userprefs.Preference objects matching the category. Returning copies ensures immutability.
// If the user ID is not found, or if no preferences match the category for that user,
// an empty map and a nil error are returned.
// This method always returns a nil error.
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

// Close is a no-op for MemoryStorage.
// Since MemoryStorage operates entirely in-memory without external resources like
// database connections or file handles, there is nothing to release or clean up.
// This method is provided to satisfy the userprefs.Storage interface.
// It always returns a nil error.
func (s *MemoryStorage) Close() error {
	return nil
}
