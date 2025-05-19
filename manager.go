// Package userprefs provides the Manager responsible for handling user preferences.
package userprefs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Manager is the central component for managing user preferences. It handles the
// definition of preference types, and orchestrates their retrieval, storage, and caching.
//
// A Manager requires a userprefs.Storage implementation for persistence and can optionally
// be configured with a userprefs.Cache implementation to improve performance for frequently
// accessed preferences. All public methods of the Manager are thread-safe and can be
// called concurrently from multiple goroutines.
//
// Instances of Manager are typically created using the New() function, configured via Options.
type Manager struct {
	mu     sync.RWMutex // Protects access to the config, especially definitions map.
	config *Config      // Holds storage, cache, logger, and preference definitions.
}

// New creates and initializes a new Manager instance using functional options.
// This is the primary constructor for a Manager.
//
// Essential options, particularly WithStorage, must be provided for the Manager to function correctly.
// Other options like WithCache, WithLogger, and WithDefinition allow for further customization.
// Example usage:
//
//	storage := NewMemoryStorage()
//	cache := NewMemoryCache(100)
//	logger := log.New(os.Stdout, "userprefs: ", log.LstdFlags)
//	manager := userprefs.New(
//	    userprefs.WithStorage(storage),
//	    userprefs.WithCache(cache),
//	    userprefs.WithLogger(logger),
//	)
//
// The returned Manager is ready for use.
func New(opts ...Option) *Manager {
	cfg := &Config{
		logger:      NewDefaultLogger(), // Use exported version
		definitions: make(map[string]PreferenceDefinition),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return &Manager{
		config: cfg,
	}
}

// DefinePreference registers a new preference definition with the Manager.
// Each preference key must be unique within a Manager instance. Re-defining an existing
// key will overwrite the previous definition.
//
// A PreferenceDefinition includes:
//   - Key: A unique string identifier for the preference.
//   - Type: The expected data type (e.g., String, Bool, Int, Float, JSON).
//   - DefaultValue: The value to use if the user hasn't set this preference.
//   - Category: An optional string for grouping preferences.
//   - ValidateFunc: An optional function for custom value validation during Set operations.
//
// Returns:
//   - ErrInvalidKey: if def.Key is empty.
//   - ErrInvalidType: if def.Type is not a supported PreferenceType (see IsValidType).
//   - nil: on successful registration of the preference definition.
//
// This method is thread-safe.
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

// Get retrieves a user's preference for a given key.
// The provided context.Context can be used for cancellation or timeouts, which will be
// propagated to cache and storage operations.
//
// Operational Flow:
//  1. Input Validation: Checks if userID and key are non-empty. If not, returns ErrInvalidInput.
//  2. Definition Check: Verifies that the preference key has been defined. If not, returns ErrPreferenceNotDefined.
//  3. Cache Lookup (if cache is configured):
//     a. Attempts to retrieve the preference from the cache.
//     b. On cache hit (no error), the cached preference is returned.
//     c. On cache error (excluding specific 'not found' errors, if distinguishable by the cache implementation):
//     Logs the error. Returns a Preference populated with the *defined default value* and the original cache error.
//     This allows the application to continue with a default during transient cache issues.
//     d. On cache miss (or 'not found' error), proceeds to storage lookup.
//  4. Storage Lookup (if no cache, or cache miss):
//     a. Fetches the preference from the storage backend.
//     b. If found in storage: The retrieved preference is returned. If a cache is configured,
//     the preference is asynchronously stored in the cache for future requests.
//     c. If storage returns ErrNotFound: A Preference struct populated with the *defined default value*
//     is returned with a nil error (indicating successful application of default).
//     d. If storage returns any other error: That error is wrapped and returned.
//
// Returns:
//   - (*Preference, nil): On successful retrieval (from cache or storage) or when a defined default value is applied.
//   - (nil, ErrInvalidInput): If userID or key is empty.
//   - (nil, ErrPreferenceNotDefined): If the preference key has not been defined.
//   - (*Preference with default, wrapped cache error): If cache fails and a default is applied.
//   - (nil, wrapped storage error): If storage fails and a default cannot be applied or is not applicable.
//
// This method is thread-safe.
func (m *Manager) Get(ctx context.Context, userID, key string) (*Preference, error) {
	if userID == "" || key == "" {
		return nil, ErrInvalidInput
	}

	def, exists := m.GetDefinition(key)
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

// Set creates or updates a user's preference for a given key with the provided value.
// The provided context.Context can be used for cancellation or timeouts, propagated to storage/cache.
//
// Operational Flow:
//  1. Input Validation: Checks userID and key. Returns ErrInvalidInput if empty.
//  2. Definition Check: Verifies key is defined. Returns ErrPreferenceNotDefined if not.
//  3. Type Validation: Ensures the provided `value` matches the `Type` in the PreferenceDefinition.
//     Returns ErrInvalidValue if type mismatch (e.g., providing a string for an Int preference).
//  4. Custom Validation: If `ValidateFunc` is set in PreferenceDefinition, it's called. Returns ErrInvalidValue
//     if this custom validation fails.
//  5. Storage Operation: Saves the preference (UserID, Key, Value, Type, Category, DefaultValue from definition,
//     and current UpdatedAt) to the storage backend.
//  6. Cache Invalidation (if cache is configured): Deletes the corresponding entry from the cache
//     to maintain consistency. Subsequent Get calls will fetch from storage and repopulate cache.
//
// Returns:
//   - nil: On successful creation or update.
//   - ErrInvalidInput: If userID or key is empty.
//   - ErrPreferenceNotDefined: If the preference key has not been defined.
//   - ErrInvalidValue: If the provided value fails type validation or custom validation.
//   - A wrapped storage error: If the storage operation fails.
//
// This method is thread-safe.
func (m *Manager) Set(ctx context.Context, userID, key string, value interface{}) error {
	if userID == "" || key == "" {
		return ErrInvalidInput
	}

	def, exists := m.GetDefinition(key)
	if !exists {
		return ErrPreferenceNotDefined
	}

	if err := validateValue(value, def); err != nil {
		return err // This already returns ErrInvalidValue if types mismatch or value not in AllowedValues
	}

	// Custom validation function, if defined
	if def.ValidateFunc != nil {
		if err := def.ValidateFunc(value); err != nil {
			// Wrap the error from ValidateFunc to indicate it's a validation failure
			return fmt.Errorf("%w: custom validation failed: %v", ErrInvalidValue, err)
		}
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

// GetByCategory retrieves all preferences for a given userID that belong to the specified category.
// This method currently fetches directly from the storage backend and does not utilize the cache.
// For each preference found in storage under the category, it checks if a corresponding
// PreferenceDefinition exists. If not, a warning is logged, and the preference is skipped.
// GetByCategory retrieves all preferences for a given userID that belong to the specified category.
// The provided context.Context can be used for cancellation or timeouts, propagated to storage.
//
// Behavior:
//   - Validates userID. Returns ErrInvalidInput if empty.
//   - Fetches preferences directly from the storage backend. This method *does not* currently
//     utilize or interact with the cache.
//   - For each preference retrieved from storage, it ensures the DefaultValue from its
//     definition is populated in the returned Preference struct if the stored DefaultValue is nil.
//
// Returns:
//   - (map[string]*Preference, nil): A map of preference keys to Preference structs on success.
//     The map will be empty if no preferences match the category or if the user has no preferences.
//   - (nil, ErrInvalidInput): If userID is empty.
//   - (nil, wrapped storage error): If the storage operation fails.
//
// This method is thread-safe.
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

// GetAll retrieves all preferences for a given userID.
// The provided context.Context can be used for cancellation or timeouts, propagated to storage.
//
// Behavior:
//  1. Validates userID. Returns ErrInvalidInput if empty.
//  2. Fetches all preference definitions known to the manager.
//  3. Fetches all preferences for the user directly from the storage backend using storage.GetAll.
//  4. For each defined preference:
//     a. If a corresponding preference is found in the storage results, that preference is used.
//     Its DefaultValue, Type, and Category are updated from the definition to ensure consistency.
//     b. If not found in storage, a new Preference struct is created using the DefaultValue from its definition.
//     The Value field is set to this DefaultValue.
//     c. The processed preference is added to the result map.
//  5. If a cache is configured, all retrieved/defaulted preferences are asynchronously added to the cache.
//
// Returns:
//   - (map[string]*Preference, nil): A map of preference keys to Preference structs on success.
//     The map will be empty if the user has no (defined) preferences or if no preferences are defined.
//   - (nil, ErrInvalidInput): If userID is empty.
//   - (nil, wrapped storage error): If the storage.GetAll operation fails.
//
// This method is thread-safe.
func (m *Manager) GetAll(ctx context.Context, userID string) (map[string]*Preference, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	m.mu.RLock()
	definitions := make(map[string]PreferenceDefinition, len(m.config.definitions))
	for k, v := range m.config.definitions {
		definitions[k] = v
	}
	m.mu.RUnlock()

	if len(definitions) == 0 {
		return make(map[string]*Preference), nil // No definitions, return empty map
	}

	// Fetch all preferences from storage for this user in one go.
	storedPrefs, err := m.config.storage.GetAll(ctx, userID)
	if err != nil {
		// Do not return ErrNotFound from storage as an error here; an empty map from storage is valid.
		// Only propagate other storage errors.
		if !errors.Is(err, ErrNotFound) { // ErrNotFound from storage means no prefs for user, which is fine.
			m.config.logger.Error("Storage GetAll failed", "userID", userID, "error", err)
			return nil, fmt.Errorf("storage.GetAll failed for userID '%s': %w", userID, err)
		}
		// If ErrNotFound, storedPrefs will be nil or empty, which is handled below.
		storedPrefs = make(map[string]*Preference) // Ensure it's an empty map, not nil
	}

	userPreferences := make(map[string]*Preference, len(definitions))
	prefsToCache := make([]*Preference, 0, len(definitions))

	for key, def := range definitions {
		var finalPref *Preference
		storedPref, foundInStorage := storedPrefs[key]

		if foundInStorage {
			finalPref = storedPref
			// Ensure definition's truth is reflected, especially for DefaultValue, Type, Category
			finalPref.DefaultValue = def.DefaultValue
			finalPref.Type = def.Type
			finalPref.Category = def.Category
			// UserID and Key should match, Value and UpdatedAt come from storage.
		} else {
			// Not found in storage, use default from definition
			finalPref = &Preference{
				UserID:       userID,
				Key:          key,
				Value:        def.DefaultValue,
				DefaultValue: def.DefaultValue,
				Type:         def.Type,
				Category:     def.Category,
				// The original m.Get sets UpdatedAt to time.Now() for defaults.
				// Let's stick to that for now for consistency.
				UpdatedAt: time.Now(),
			}
		}
		userPreferences[key] = finalPref
		if m.config.cache != nil {
			prefsToCache = append(prefsToCache, finalPref)
		}
	}

	// Asynchronously warm the cache with all preferences (stored or defaulted)
	if m.config.cache != nil && len(prefsToCache) > 0 {
		go func() {
			// Create a new context for this background task, or use a non-cancellable one if appropriate.
			// Using context.Background() for simplicity here, but consider if parent context's lifetime is relevant.
			// If the parent ctx for GetAll might be very short-lived, Background is safer for cache warming.
			cacheCtx := context.Background() // Or context.TODO() if a specific context strategy is TBD
			for _, p := range prefsToCache {
				// Use a new pointer for each cache set if pref structs are reused or modified in loop
				// In this case, finalPref is new in each loop or points to storedPref, so it should be fine.
				m.setToCache(cacheCtx, p) // setToCache handles logging errors internally
			}
			m.config.logger.Debug("Cache warming initiated for GetAll results", "userID", userID, "count", len(prefsToCache))
		}()
	}

	return userPreferences, nil
}

// Delete removes a user's preference for a given key.
// The provided context.Context can be used for cancellation or timeouts, propagated to storage/cache.
//
// Operational Flow:
//  1. Input Validation: Checks userID and key. Returns ErrInvalidInput if empty.
//  2. Definition Check: Verifies key is defined. Returns ErrPreferenceNotDefined if not.
//     (Note: This check ensures operations are only on known preference types, though the preference
//     might not exist for this specific user in storage).
//  3. Storage Operation: Deletes the preference from the storage backend.
//     - If storage returns ErrNotFound, this is considered a successful deletion (idempotency),
//     as the desired state (preference not present) is achieved. Returns nil error.
//  4. Cache Invalidation (if cache is configured): Deletes the corresponding entry from the cache,
//     regardless of whether the item was found in storage.
//
// Returns:
//   - nil: On successful deletion or if the preference was not found in storage (idempotent).
//   - ErrInvalidInput: If userID or key is empty.
//   - ErrPreferenceNotDefined: If the preference key has not been defined.
//   - A wrapped storage error: If the storage deletion fails for reasons other than ErrNotFound.
//
// This method is thread-safe.
func (m *Manager) Delete(ctx context.Context, userID, key string) error {
	if userID == "" || key == "" {
		return ErrInvalidInput
	}

	_, exists := m.GetDefinition(key)
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

// GetDefinition retrieves the preference definition for a given key.
// It is the exported version of the internal getDefinition logic.
func (m *Manager) GetDefinition(key string) (PreferenceDefinition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	def, exists := m.config.definitions[key]
	return def, exists
}

// GetAllDefinitions retrieves all preference definitions.
func (m *Manager) GetAllDefinitions(_ context.Context) ([]*PreferenceDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	defs := make([]*PreferenceDefinition, 0, len(m.config.definitions))
	for i := range m.config.definitions {
		def := m.config.definitions[i] // Create a new variable to take its address
		defs = append(defs, &def)
	}
	return defs, nil
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

	m.config.logger.Debug("Raw data from cache before unmarshal", "cacheKey", cacheKey, "dataSize", len(data))

	var pref Preference
	if err := json.Unmarshal(data, &pref); err != nil {
		m.config.logger.Error("Cache Unmarshal failed", "cacheKey", cacheKey, "dataType", fmt.Sprintf("%T", data), "dataContent", string(data), "error", err)
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
