// types.go
package userprefs

import (
	"time"
)

type Preference struct {
	UserID       string      `json:"user_id"`
	Key          string      `json:"key"`
	Value        interface{} `json:"value"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Type         string      `json:"type"`
	Category     string      `json:"category,omitempty"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type PreferenceDefinition struct {
	Key           string        `json:"key"`
	Type          string        `json:"type"`
	DefaultValue  interface{}   `json:"default_value,omitempty"`
	Category      string        `json:"category,omitempty"`
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
}

type Config struct {
	storage    Storage
	cache      Cache
	logger     Logger
	definitions map[string]PreferenceDefinition
}

type Option func(*Config)

func WithStorage(s Storage) Option {
	return func(c *Config) {
		c.storage = s
	}
}

func WithCache(cache Cache) Option {
	return func(c *Config) {
		c.cache = cache
	}
}

func WithLogger(l Logger) Option {
	return func(c *Config) {
		c.logger = l
	}
}