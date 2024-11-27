package userprefs

import (
	"testing"
	"time"
)

func TestPreferenceStruct(t *testing.T) {
	pref := Preference{
		UserID:       "user1",
		Key:          "theme",
		Value:        "dark",
		DefaultValue: "light",
		Type:         "enum",
		Category:     "appearance",
		UpdatedAt:    time.Now(),
	}

	if pref.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", pref.UserID)
	}
	if pref.Key != "theme" {
		t.Errorf("Expected Key 'theme', got '%s'", pref.Key)
	}
	if pref.Value != "dark" {
		t.Errorf("Expected Value 'dark', got '%v'", pref.Value)
	}
	if pref.DefaultValue != "light" {
		t.Errorf("Expected DefaultValue 'light', got '%v'", pref.DefaultValue)
	}
	if pref.Type != "enum" {
		t.Errorf("Expected Type 'enum', got '%s'", pref.Type)
	}
	if pref.Category != "appearance" {
		t.Errorf("Expected Category 'appearance', got '%s'", pref.Category)
	}
	if pref.UpdatedAt.IsZero() {
		t.Errorf("Expected UpdatedAt to be set, got zero value")
	}
}

func TestPreferenceDefinitionStruct(t *testing.T) {
	def := PreferenceDefinition{
		Key:           "language",
		Type:          "string",
		DefaultValue:  "en",
		Category:      "general",
		AllowedValues: nil,
	}

	if def.Key != "language" {
		t.Errorf("Expected Key 'language', got '%s'", def.Key)
	}
	if def.Type != "string" {
		t.Errorf("Expected Type 'string', got '%s'", def.Type)
	}
	if def.DefaultValue != "en" {
		t.Errorf("Expected DefaultValue 'en', got '%v'", def.DefaultValue)
	}
	if def.Category != "general" {
		t.Errorf("Expected Category 'general', got '%s'", def.Category)
	}
	if def.AllowedValues != nil {
		t.Errorf("Expected AllowedValues nil, got '%v'", def.AllowedValues)
	}
}

func TestConfigStruct(t *testing.T) {
	store := &MockStorage{}
	cache := &MockCache{}
	logger := &MockLogger{}
	definitions := make(map[string]PreferenceDefinition)

	cfg := Config{
		storage:     store,
		cache:       cache,
		logger:      logger,
		definitions: definitions,
	}

	if cfg.storage != store {
		t.Errorf("Expected storage to be set")
	}
	if cfg.cache != cache {
		t.Errorf("Expected cache to be set")
	}
	if cfg.logger != logger {
		t.Errorf("Expected logger to be set")
	}
	if len(cfg.definitions) != 0 {
		t.Errorf("Expected definitions to be empty")
	}
}

func TestOptionFunctions(t *testing.T) {
	store := &MockStorage{}
	cache := &MockCache{}
	logger := &MockLogger{}
	// definitions := make(map[string]PreferenceDefinition)

	cfg := Config{
		storage:     nil,
		cache:       nil,
		logger:      nil,
		definitions: nil,
	}

	WithStorage(store)(&cfg)
	if cfg.storage != store {
		t.Errorf("WithStorage failed to set storage")
	}

	WithCache(cache)(&cfg)
	if cfg.cache != cache {
		t.Errorf("WithCache failed to set cache")
	}

	WithLogger(logger)(&cfg)
	if cfg.logger != logger {
		t.Errorf("WithLogger failed to set logger")
	}
}
