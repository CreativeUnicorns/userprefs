package userprefs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// MockStorage implements the Storage interface for testing
type MockStorage struct {
	mu               sync.RWMutex
	data             map[string]map[string]*Preference
	closed           bool
	forceGetByCatErr error // For forcing errors in GetByCategory for testing
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string]map[string]*Preference),
	}
}

func (m *MockStorage) Get(ctx context.Context, userID, key string) (*Preference, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageUnavailable
	}

	if userPrefs, exists := m.data[userID]; exists {
		if pref, exists := userPrefs[key]; exists {
			// Return a deep copy to prevent modifications from affecting stored data
			copiedPref, err := deepCopyPreference(pref)
			if err != nil {
				return nil, fmt.Errorf("mockstorage: error deep copying preference %s for user %s: %w", key, userID, err)
			}
			return copiedPref, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MockStorage) Set(ctx context.Context, pref *Preference) error {
	_, _ = ctx.Deadline()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageUnavailable
	}

	if _, exists := m.data[pref.UserID]; !exists {
		m.data[pref.UserID] = make(map[string]*Preference)
	}
	m.data[pref.UserID][pref.Key] = pref
	return nil
}

func (m *MockStorage) Delete(ctx context.Context, userID, key string) error {
	_, _ = ctx.Deadline()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrStorageUnavailable
	}

	if userPrefs, exists := m.data[userID]; exists {
		if _, exists := userPrefs[key]; exists {
			delete(userPrefs, key)
			return nil
		}
	}
	return ErrNotFound
}

func (m *MockStorage) GetAll(ctx context.Context, userID string) (map[string]*Preference, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageUnavailable
	}

	if userPrefs, exists := m.data[userID]; exists && len(userPrefs) > 0 {
		copiedPrefs := make(map[string]*Preference, len(userPrefs))
		for key, p := range userPrefs {
			copiedP, err := deepCopyPreference(p) // Use the actual Preference type from the userprefs package
			if err != nil {
				return nil, fmt.Errorf("mockstorage: error deep copying preference %s for user %s: %w", key, userID, err)
			}
			copiedPrefs[key] = copiedP
		}
		return copiedPrefs, nil
	}
	// Return empty map instead of error when no preferences exist for the user
	return make(map[string]*Preference), nil
}

// SetGetByCategoryError sets up an error to be returned by GetByCategory
func (m *MockStorage) SetGetByCategoryError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceGetByCatErr = err
}

func (m *MockStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageUnavailable
	}

	// Return forced error if set
	if m.forceGetByCatErr != nil {
		return nil, m.forceGetByCatErr
	}

	result := make(map[string]*Preference)
	if userPrefs, exists := m.data[userID]; exists {
		for key, pref := range userPrefs {
			if pref.Category == category {
				copiedP, err := deepCopyPreference(pref) // Use the actual Preference type
				if err != nil {
					return nil, fmt.Errorf("mockstorage: error deep copying preference %s for user %s (category %s): %w", pref.Key, userID, category, err)
				}
				result[key] = copiedP
			}
		}
	}

	// Return empty map instead of error when no preferences exist for the category
	return result, nil
}

func (m *MockStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// deepCopyInterface performs a deep copy of an interface{} type by marshalling and unmarshalling it.
// This is effective for JSON-compatible data types.
func deepCopyInterface(data interface{}) (interface{}, error) {
	if data == nil {
		return nil, nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("deepCopyInterface: failed to marshal: %w", err)
	}
	var copiedData interface{}
	if err := json.Unmarshal(bytes, &copiedData); err != nil {
		return nil, fmt.Errorf("deepCopyInterface: failed to unmarshal: %w", err)
	}
	return copiedData, nil
}

// deepCopyPreference creates a deep copy of a Preference object.
// Note: This function should use the actual Preference type from the userprefs package.
// For this example, it's assumed to be userprefs.Preference.
func deepCopyPreference(original *Preference) (*Preference, error) {
	if original == nil {
		return nil, nil
	}

	copiedValue, err := deepCopyInterface(original.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy Value for key %s: %w", original.Key, err)
	}

	copiedDefaultValue, err := deepCopyInterface(original.DefaultValue)
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy DefaultValue for key %s: %w", original.Key, err)
	}

	// Create a new Preference struct and copy values.
	// For time.Time, direct assignment is a copy of the value.
	return &Preference{
		UserID:       original.UserID,
		Key:          original.Key,
		Value:        copiedValue,
		DefaultValue: copiedDefaultValue,
		Type:         original.Type,
		Category:     original.Category,
		UpdatedAt:    original.UpdatedAt, // time.Time is a struct, direct assignment copies its value.
	}, nil
}

// mockCacheEntry holds a value and an error for a cache key.
// This allows tests to pre-configure specific return values and errors for MockCache.Get.
type mockCacheEntry struct {
	value []byte
	err   error
}

// MockCache implements the Cache interface for testing
type MockCache struct {
	mu     sync.RWMutex
	data   map[string]mockCacheEntry // Stores mockCacheEntry instead of raw interface{}
	closed bool
}

// NewMockCache creates a new MockCache for testing.
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]mockCacheEntry),
	}
}

// Get retrieves a value from the mock cache. It returns the pre-configured value and error
// for the key if an entry exists. If an error is set in the entry, that error is returned.
// If the key is not found, it returns (nil, ErrNotFound).
func (m *MockCache) Get(ctx context.Context, key string) ([]byte, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrCacheUnavailable
	}

	entry, exists := m.data[key]
	if !exists {
		return nil, ErrNotFound // Standard cache miss
	}

	if entry.err != nil {
		return entry.value, entry.err // Return pre-configured error (value might also be set for some test cases)
	}

	return entry.value, nil
}

// Set stores a value in the mock cache. It wraps the value in a mockCacheEntry.
// To simulate Set returning an error, the 'closed' flag can be used, or this method
// could be extended if more complex error simulations for Set are needed.
func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	_, _ = ctx.Deadline()
	_ = ttl // TTL is ignored in this mock implementation

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrCacheUnavailable
	}

	// Store as a mockCacheEntry with no error, as Set itself is successful.
	// Errors for Get are configured by directly manipulating the 'data' map if needed for specific test cases,
	// or by enhancing 'Set' to accept an error to be returned by subsequent 'Get's.
	m.data[key] = mockCacheEntry{value: value, err: nil}
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	_, _ = ctx.Deadline()
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrCacheUnavailable
	}

	if _, exists := m.data[key]; exists {
		delete(m.data, key)
		return nil
	}
	return ErrNotFound
}

func (m *MockCache) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// MockLogger implements the Logger interface for testing
type MockLogger struct {
	mu       sync.Mutex
	Messages []string
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, formatMessage("DEBUG", msg, args...))
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, formatMessage("INFO", msg, args...))
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, formatMessage("WARN", msg, args...))
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, fmt.Sprintf("ERROR: "+msg, args...))
}

// SetLevel is a mock implementation of the SetLevel method.
// It records the attempt to set the log level for test verification.
func (m *MockLogger) SetLevel(level LogLevel) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, fmt.Sprintf("SET_LEVEL: %v", level))
}

func formatMessage(level, msg string, args ...interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf("%s: %s %v", level, msg, args)
	}
	return fmt.Sprintf("%s: %s", level, msg)
}
