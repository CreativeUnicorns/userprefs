// storage/storage.go
package storage

import (
	"context"

	"github.com/CreativeUnicorns/userprefs"
)

type Storage interface {
	Get(ctx context.Context, userID, key string) (*userprefs.Preference, error)
	Set(ctx context.Context, pref *userprefs.Preference) error
	Delete(ctx context.Context, userID, key string) error
	GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error)
	GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error)
	Close() error
}
