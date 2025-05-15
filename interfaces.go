// Package userprefs defines interfaces for storage, caching, and logging used in user preferences management.
package userprefs

import (
	"context"
	"time"
)

// Storage defines the methods required for a storage backend.
type Storage interface {
	Get(ctx context.Context, userID, key string) (*Preference, error)
	Set(ctx context.Context, pref *Preference) error
	Delete(ctx context.Context, userID, key string) error
	GetAll(ctx context.Context, userID string) (map[string]*Preference, error)
	GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error)
	Close() error
}

// Cache defines the methods required for a caching backend.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

// Logger defines the methods required for logging within the user preferences system.
// The args should be alternating key-value pairs, similar to slog.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
