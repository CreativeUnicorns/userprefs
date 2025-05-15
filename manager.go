// Package userprefs provides the Manager responsible for handling user preferences.
package userprefs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Manager manages user preferences by interfacing with storage and cache backends.
type Manager struct {
	mu     sync.RWMutex
	config *Config
}

// New creates a new Manager instance with the provided configuration options.
func New(opts ...Option) *Manager {
	cfg := &Config{
		logger:      newDefaultLogger(),
		definitions: make(map[string]PreferenceDefinition),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &Manager{
		config: cfg,
	}
}

// DefinePreference registers a new preference definition.
// It returns an error if the key is invalid or the type is unsupported.
func (m *Manager) DefinePreference(def PreferenceDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if def.Key == "" {
		return ErrInvalidKey
	}

	if !isValidType(def.Type) {
		return ErrInvalidType
	}

	m.config.definitions[def.Key] = def
	return nil
}

// Get retrieves a preference for a user by key.
// It returns the preference or an error if not found.
func (m *Manager) Get(ctx context.Context, userID, key string) (*Preference, error) {
	if userID == "" || key == "" {
		return nil, ErrInvalidInput
	}

	def, exists := m.getDefinition(key)
	if !exists {
		return nil, ErrPreferenceNotDefined
	}

	if m.config.cache != nil {
		prefFromCache, cacheErr := m.getFromCache(ctx, userID, key)
		if cacheErr == nil { // Cache hit, no error
			m.config.logger.Debug("Cache hit", "userID", userID, "key", key)
			return prefFromCache, nil
		}

		// Cache returned an error. Log it.
		m.config.logger.Warn("Failed to get preference from cache", "userID", userID, "key", key, "error", cacheErr)

		// If the cache error is NOT a simple 'ErrNotFound' (cache miss),
		// it's a more significant issue (e.g., serialization, or a deeper cache system error).
		// In this case, return the default value along with this cache error.
		if !errors.Is(cacheErr, ErrNotFound) {
			m.config.logger.Error("Cache error is not ErrNotFound, returning default and propagating cache error", "userID", userID, "key", key, "originalError", cacheErr)
			return &Preference{
				UserID:       userID,
				Key:          key,
				Value:        def.DefaultValue,
				DefaultValue: def.DefaultValue,
				Type:         def.Type,
				Category:     def.Category,
				UpdatedAt:    time.Now(), // Consider if a zero time or definition's timestamp is more appropriate
			}, cacheErr // Propagate the actual cache error
		}
		// If errors.Is(cacheErr, ErrNotFound), it was a clean cache miss. Proceed to storage.
		m.config.logger.Debug("Cache miss (ErrNotFound from cache layer), proceeding to storage", "userID", userID, "key", key)
	}

	// Fallback to storage if cache is nil or if getFromCache resulted in ErrNotFound.
	m.config.logger.Debug("Fetching from storage", "userID", userID, "key", key)
	pref, err := m.config.storage.Get(ctx, userID, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) { // Use errors.Is for checking predefined errors
			// If not found in storage, return the preference with its default value
			return &Preference{
				UserID:       userID,
				Key:          key,
				Value:        def.DefaultValue,
				DefaultValue: def.DefaultValue, // Ensure DefaultValue is also populated here
				Type:         def.Type,
				Category:     def.Category,
				UpdatedAt:    time.Now(), // Or perhaps a zero time if it's purely a default?
			}, nil
		}
		m.config.logger.Error("Storage Get failed", "userID", userID, "key", key, "error", err)
		return nil, fmt.Errorf("storage.Get failed for key '%s': %w", key, err)
	}

	if m.config.cache != nil {
		m.setToCache(ctx, pref)
	}

	return pref, nil
}

// Set updates or creates a preference for a user.
// It validates the value against the preference definition.
func (m *Manager) Set(ctx context.Context, userID, key string, value interface{}) error {
	if userID == "" || key == "" {
		return ErrInvalidInput
	}

	def, exists := m.getDefinition(key)
	if !exists {
		return ErrPreferenceNotDefined
	}

	if err := validateValue(value, def); err != nil {
		return err
	}

	pref := &Preference{
		UserID:       userID,
		Key:          key,
		Value:        value,
		DefaultValue: def.DefaultValue, // Added this line
		Type:         def.Type,
		Category:     def.Category,
		UpdatedAt:    time.Now(),
	}

	if err := m.config.storage.Set(ctx, pref); err != nil {
		m.config.logger.Error("Storage Set failed", "userID", userID, "key", key, "error", err)
		return fmt.Errorf("storage.Set failed for key '%s': %w", key, err)
	}

	if m.config.cache != nil {
		m.setToCache(ctx, pref)
	}

	return nil
}

// GetByCategory retrieves all preferences for a user within a specific category.
func (m *Manager) GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error) {
	if userID == "" || category == "" {
		return nil, ErrInvalidInput
	}

	prefs, err := m.config.storage.GetByCategory(ctx, userID, category)
	if err != nil {
		m.config.logger.Error("Storage GetByCategory failed", "userID", userID, "category", category, "error", err)
		return nil, fmt.Errorf("storage.GetByCategory failed for category '%s': %w", category, err)
	}
	return prefs, nil
}

// GetAll retrieves all preferences for a user.
func (m *Manager) GetAll(ctx context.Context, userID string) (map[string]*Preference, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	m.mu.RLock()
	definedKeys := make([]string, 0, len(m.config.definitions))
	for k := range m.config.definitions {
		definedKeys = append(definedKeys, k)
	}
	m.mu.RUnlock()

	if len(definedKeys) == 0 {
		return make(map[string]*Preference), nil // No definitions, return empty map
	}

	userPreferences := make(map[string]*Preference)
	for _, key := range definedKeys {
		pref, err := m.Get(ctx, userID, key) // m.Get handles defaults
		if err != nil {
			// If m.Get returns ErrPreferenceNotDefined here, it's an inconsistency,
			// as we are iterating over keys confirmed to be defined moments ago.
			// Other errors (e.g. storage issue) should be propagated.
			if errors.Is(err, ErrPreferenceNotDefined) {
				m.config.logger.Error("Internal inconsistency in GetAll: defined preference not found by Get", "key", key, "userID", userID)
				// This error is already logged and wrapped by m.Get or this specific block
				return nil, fmt.Errorf("internal error retrieving defined preference '%s' for GetAll: %w", key, err)
			}
			// Errors from m.Get are already wrapped, pass them up.
			// Log here for GetAll context if desired, though m.Get likely logged it too.
			m.config.logger.Warn("Failed to get preference during GetAll operation", "userID", userID, "key", key, "error", err)
			return nil, fmt.Errorf("failed to get preference '%s' for GetAll: %w", key, err)
		}
		// m.Get should always return a non-nil pref if error is nil (it provides default value from definition)
		if pref == nil {
			// This state should ideally not be reached given m.Get's contract
			m.config.logger.Error("Unexpected nil preference from m.Get without error in GetAll", "key", key, "userID", userID, "context", "GetAll")
			return nil, fmt.Errorf("%w: m.Get returned nil preference for key '%s' without an error during GetAll", ErrInternal, key)
		}
		userPreferences[key] = pref
	}

	return userPreferences, nil
}

// Delete removes a preference for a user by key.
func (m *Manager) Delete(ctx context.Context, userID, key string) error {
	if userID == "" || key == "" {
		return ErrInvalidInput
	}

	_, exists := m.getDefinition(key)
	if !exists {
		return ErrPreferenceNotDefined
	}

	if err := m.config.storage.Delete(ctx, userID, key); err != nil {
		// If storage.Delete returns ErrNotFound, it means the item was already gone
		// or never set for this user, which is fine after definition check.
		if !errors.Is(err, ErrNotFound) {
			m.config.logger.Error("Storage Delete failed", "userID", userID, "key", key, "error", err)
			return fmt.Errorf("storage.Delete failed for key '%s': %w", key, err)
		}
		// If ErrNotFound, it's okay, the item wasn't there to delete or already deleted.
	}

	if m.config.cache != nil {
		m.deleteFromCache(ctx, userID, key)
	}

	return nil
}

// getDefinition retrieves the preference definition for a given key.
func (m *Manager) getDefinition(key string) (PreferenceDefinition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	def, exists := m.config.definitions[key]
	return def, exists
}

// getFromCache retrieves a preference from the cache.
func (m *Manager) getFromCache(ctx context.Context, userID, key string) (*Preference, error) {
	cacheKey := fmt.Sprintf("pref:%s:%s", userID, key)
	data, err := m.config.cache.Get(ctx, cacheKey)
	if err != nil {
		// Don't log simple cache misses if cache returns a specific 'not found' error.
		// Assuming any other error is unexpected for getFromCache.
		// if !errors.Is(err, cache.ErrNotFound) { // Assuming cache might have its own ErrNotFound
		// 	 m.config.logger.Warn("Cache Get failed", "cacheKey", cacheKey, "error", err)
		// }
		return nil, fmt.Errorf("cache.Get failed for key '%s': %w", cacheKey, err) // Wrap for now
	}

	dataBytes, ok := data.([]byte)
	if !ok {
		m.config.logger.Error("Cache Get returned non-[]byte type", "cacheKey", cacheKey, "actualType", fmt.Sprintf("%T", data))
		return nil, fmt.Errorf("%w: cache returned non-[]byte data for key '%s' (type: %T)", ErrSerialization, cacheKey, data)
	}

	m.config.logger.Debug("Raw data from cache before unmarshal", "cacheKey", cacheKey, "dataString", string(dataBytes))

	// Attempt to handle if dataBytes is a JSON string literal containing Base64 encoded JSON
	var tempStrForBase64 string
	if unmarshalStrErr := json.Unmarshal(dataBytes, &tempStrForBase64); unmarshalStrErr == nil {
		// Successfully unmarshalled dataBytes into a string, e.g., tempStrForBase64 holds the Base64 content.
		decodedBytes, b64Err := base64.StdEncoding.DecodeString(tempStrForBase64)
		if b64Err == nil {
			// Successfully Base64 decoded. These should be the actual Preference JSON bytes.
			dataBytes = decodedBytes
			m.config.logger.Debug("Successfully decoded Base64 content from cache", "cacheKey", cacheKey, "decodedDataString", string(dataBytes))
		} else {
			// Failed to Base64 decode, log and proceed with original dataBytes (which will likely fail unmarshal).
			m.config.logger.Warn("Cache data looked like JSON string, but failed Base64 decoding", "cacheKey", cacheKey, "base64StringAttempted", tempStrForBase64, "error", b64Err)
		}
	} // else, dataBytes was not a simple JSON string; assume it's the direct JSON object bytes (or will fail unmarshal).

	var pref Preference
	if err := json.Unmarshal(dataBytes, &pref); err != nil {
		m.config.logger.Error("Cache Unmarshal failed", "cacheKey", cacheKey, "dataType", fmt.Sprintf("%T", dataBytes), "dataContent", string(dataBytes), "error", err)
		return nil, fmt.Errorf("%w: failed to unmarshal cached preference for key '%s': %v", ErrSerialization, cacheKey, err)
	}

	return &pref, nil
}

// setToCache stores a preference in the cache.
func (m *Manager) setToCache(ctx context.Context, pref *Preference) {
	cacheKey := fmt.Sprintf("pref:%s:%s", pref.UserID, pref.Key)
	data, err := json.Marshal(pref)
	if err != nil {
		m.config.logger.Error("Failed to marshal preference for cache", "userID", pref.UserID, "key", pref.Key, "type", ErrSerialization, "error", err)
		return
	}

	if err := m.config.cache.Set(ctx, cacheKey, data, 24*time.Hour); err != nil {
		m.config.logger.Error("Failed to cache preference", "error", err)
	}
}

// deleteFromCache removes a preference from the cache.
func (m *Manager) deleteFromCache(ctx context.Context, userID, key string) {
	cacheKey := fmt.Sprintf("pref:%s:%s", userID, key)
	if err := m.config.cache.Delete(ctx, cacheKey); err != nil {
		// Similarly, don't spam logs for misses if cache.Delete returns a specific 'not found' error.
		m.config.logger.Warn("Failed to delete preference from cache", "cacheKey", cacheKey, "error", err)
	}
}
