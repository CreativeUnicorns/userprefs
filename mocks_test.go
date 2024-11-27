package userprefs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MockStorage implements the Storage interface for testing
type MockStorage struct {
	mu     sync.RWMutex
	data   map[string]map[string]*Preference
	closed bool
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
			return pref, nil
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

	if userPrefs, exists := m.data[userID]; exists {
		return userPrefs, nil
	}
	return nil, ErrNotFound
}

func (m *MockStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrStorageUnavailable
	}

	result := make(map[string]*Preference)
	if userPrefs, exists := m.data[userID]; exists {
		for key, pref := range userPrefs {
			if pref.Category == category {
				result[key] = pref
			}
		}
	}

	if len(result) == 0 {
		return nil, ErrNotFound
	}
	return result, nil
}

func (m *MockStorage) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// MockCache implements the Cache interface for testing
type MockCache struct {
	mu     sync.RWMutex
	data   map[string]interface{}
	closed bool
}

func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
	}
}

func (m *MockCache) Get(ctx context.Context, key string) (interface{}, error) {
	_, _ = ctx.Deadline()
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, ErrCacheUnavailable
	}

	val, exists := m.data[key]
	if !exists {
		return nil, ErrNotFound
	}
	return val, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	_, _ = ctx.Deadline()
	_ = ttl

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrCacheUnavailable
	}

	m.data[key] = value
	// TTL handling can be simulated if necessary
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

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = append(m.Messages, formatMessage("ERROR", msg, args...))
}

func formatMessage(level, msg string, args ...interface{}) string {
	if len(args) > 0 {
		return fmt.Sprintf("%s: %s %v", level, msg, args)
	}
	return fmt.Sprintf("%s: %s", level, msg)
}
