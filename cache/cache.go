package cache

import (
	"context"
	"errors"
	"time"
)

// Cache defines the essential methods for a caching backend.
// Implementations of this interface, such as MemoryCache or RedisCache found within this package,
// are designed to fulfill the contract of the userprefs.Cache interface (defined in the parent package)
// for use by the userprefs.Manager.
// Implementations must be thread-safe for concurrent access.
type Cache interface {
	// Get retrieves an item from the cache by its key.
	// It returns the cached item as a byte slice ([]byte) and nil on success.
	// If the key is not found or has expired, implementations should return userprefs.ErrNotFound (from the parent package)
	// to comply with the userprefs.Cache interface contract.
	// Other errors, like cache.ErrCacheFailure, may be returned for operational issues.
	Get(ctx context.Context, key string) ([]byte, error)

	// Set adds an item to the cache with a specific key and time-to-live (TTL).
	// The 'value' parameter is the byte slice item to cache.
	// The 'ttl' parameter specifies the duration for which the item should be cached.
	// A TTL of 0 might be interpreted by implementations as "cache forever" or "use default TTL",
	// refer to specific implementation documentation for details.
	// Returns an error (e.g., cache.ErrCacheFailure) if the operation fails.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes an item from the cache by its key.
	// It should be idempotent, returning nil error even if the key does not exist in the cache.
	// Returns an error (e.g., cache.ErrCacheFailure) if the deletion attempt fails due to an operational issue.
	Delete(ctx context.Context, key string) error

	// Close releases any resources (like network connections or background goroutines)
	// held by the cache implementation. It should be called when the cache is no longer needed.
	Close() error
}

// Define cache-specific errors. Note that for broader compatibility within the userprefs library,
// implementations are encouraged to use errors defined in the parent `userprefs` package where appropriate (e.g., userprefs.ErrNotFound).
var (
	// ErrNotFound indicates a key was not found in the cache.
	// DEPRECATED: Implementations of userprefs.Cache should return userprefs.ErrNotFound (from the parent package) instead
	// to ensure consistency with the userprefs.Cache interface contract.
	ErrNotFound = errors.New("cache: key not found")

	// ErrKeyExpired indicates a key was found in the cache but has passed its expiration time.
	// Implementations might choose to return this, or treat expired keys as not found (thus returning userprefs.ErrNotFound).
	ErrKeyExpired = errors.New("cache: key expired")

	// ErrCacheFailure indicates a generic operational failure within the cache backend
	// (e.g., connection error for Redis, unexpected error for in-memory cache).
	ErrCacheFailure = errors.New("cache: operation failure")
)
