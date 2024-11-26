// manager.go
package userprefs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Manager struct {
	mu     sync.RWMutex
	config *Config
}

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

func (m *Manager) Get(ctx context.Context, userID, key string) (*Preference, error) {
	if userID == "" || key == "" {
		return nil, ErrInvalidInput
	}

	def, exists := m.getDefinition(key)
	if !exists {
		return nil, ErrPreferenceNotDefined
	}

	if m.config.cache != nil {
		if pref, err := m.getFromCache(ctx, userID, key); err == nil {
			return pref, nil
		}
	}

	pref, err := m.config.storage.Get(ctx, userID, key)
	if err != nil {
		if err == ErrNotFound {
			return &Preference{
				UserID:    userID,
				Key:       key,
				Value:     def.DefaultValue,
				Type:      def.Type,
				Category:  def.Category,
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, err
	}

	if m.config.cache != nil {
		m.setToCache(ctx, pref)
	}

	return pref, nil
}

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
		UserID:    userID,
		Key:       key,
		Value:     value,
		Type:      def.Type,
		Category:  def.Category,
		UpdatedAt: time.Now(),
	}

	if err := m.config.storage.Set(ctx, pref); err != nil {
		return err
	}

	if m.config.cache != nil {
		m.setToCache(ctx, pref)
	}

	return nil
}

func (m *Manager) GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error) {
	if userID == "" || category == "" {
		return nil, ErrInvalidInput
	}

	return m.config.storage.GetByCategory(ctx, userID, category)
}

func (m *Manager) GetAll(ctx context.Context, userID string) (map[string]*Preference, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	return m.config.storage.GetAll(ctx, userID)
}

func (m *Manager) Delete(ctx context.Context, userID, key string) error {
	if userID == "" || key == "" {
		return ErrInvalidInput
	}

	if err := m.config.storage.Delete(ctx, userID, key); err != nil {
		return err
	}

	if m.config.cache != nil {
		m.deleteFromCache(ctx, userID, key)
	}

	return nil
}

func (m *Manager) getDefinition(key string) (PreferenceDefinition, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	def, exists := m.config.definitions[key]
	return def, exists
}

func (m *Manager) getFromCache(ctx context.Context, userID, key string) (*Preference, error) {
	cacheKey := fmt.Sprintf("pref:%s:%s", userID, key)
	data, err := m.config.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var pref Preference
	if err := json.Unmarshal(data.([]byte), &pref); err != nil {
		return nil, err
	}

	return &pref, nil
}

func (m *Manager) setToCache(ctx context.Context, pref *Preference) {
	cacheKey := fmt.Sprintf("pref:%s:%s", pref.UserID, pref.Key)
	data, err := json.Marshal(pref)
	if err != nil {
		m.config.logger.Error("Failed to marshal preference for cache", "error", err)
		return
	}

	if err := m.config.cache.Set(ctx, cacheKey, data, 24*time.Hour); err != nil {
		m.config.logger.Error("Failed to cache preference", "error", err)
	}
}

func (m *Manager) deleteFromCache(ctx context.Context, userID, key string) {
	cacheKey := fmt.Sprintf("pref:%s:%s", userID, key)
	if err := m.config.cache.Delete(ctx, cacheKey); err != nil {
		m.config.logger.Error("Failed to delete preference from cache", "error", err)
	}
}
