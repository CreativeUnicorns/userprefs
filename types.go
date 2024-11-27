// Package userprefs defines the core types used in the user preferences management system.
package userprefs

import (
	"time"
)

// Preference represents a single user preference.
type Preference struct {
	UserID       string      `json:"user_id"`
	Key          string      `json:"key"`
	Value        interface{} `json:"value"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Type         string      `json:"type"`
	Category     string      `json:"category,omitempty"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// PreferenceDefinition defines the schema for a preference, including its type and allowed values.
type PreferenceDefinition struct {
	Key           string        `json:"key"`
	Type          string        `json:"type"`
	DefaultValue  interface{}   `json:"default_value,omitempty"`
	Category      string        `json:"category,omitempty"`
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
}

// Config holds the configuration for the Manager, including storage, cache, logger, and preference definitions.
type Config struct {
	storage     Storage
	cache       Cache
	logger      Logger
	definitions map[string]PreferenceDefinition
}

// Option defines a configuration option for the Manager.
type Option func(*Config)

// WithStorage sets the storage backend for the Manager.
func WithStorage(s Storage) Option {
	return func(c *Config) {
		c.storage = s
	}
}

// WithCache sets the caching backend for the Manager.
func WithCache(cache Cache) Option {
	return func(c *Config) {
		c.cache = cache
	}
}

// WithLogger sets the logger for the Manager.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		c.logger = l
	}
}
