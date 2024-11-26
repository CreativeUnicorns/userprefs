// interfaces.go
package userprefs

import (
    "context"
    "time"
)

type Storage interface {
    Get(ctx context.Context, userID, key string) (*Preference, error)
    Set(ctx context.Context, pref *Preference) error
    Delete(ctx context.Context, userID, key string) error
    GetAll(ctx context.Context, userID string) (map[string]*Preference, error)
    GetByCategory(ctx context.Context, userID, category string) (map[string]*Preference, error)
    Close() error
}

type Cache interface {
    Get(ctx context.Context, key string) (interface{}, error)
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Close() error
}

type Logger interface {
    Debug(msg string, args ...interface{})
    Info(msg string, args ...interface{})
    Warn(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}