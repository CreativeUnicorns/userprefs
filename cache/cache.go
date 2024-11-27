package cache

import (
	"context"
	"errors"
	"time"
)

// Cache defines the methods required for a caching backend.
type Cache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Close() error
}

// Define cache-specific errors
var (
	ErrNotFound     = errors.New("key not found")
	ErrKeyExpired   = errors.New("key expired")
	ErrCacheFailure = errors.New("cache failure")
)
