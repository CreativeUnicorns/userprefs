// Package storage defines the Storage interface for user preferences.
package storage

import (
	"context"

	"github.com/CreativeUnicorns/userprefs"
)

// Storage defines the methods required for a storage backend.
type Storage interface {
	Get(ctx context.Context, userID, key string) (*userprefs.Preference, error)
	Set(ctx context.Context, pref *userprefs.Preference) error
	Delete(ctx context.Context, userID, key string) error
	GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error)
	GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error)
	Close() error
}
