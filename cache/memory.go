// Package cache provides in-memory caching implementations.
package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CreativeUnicorns/userprefs"
)

// item represents a single cache item with a value (as a byte slice) and an expiration time.
type item struct {
	value      []byte
	expiration time.Time
}

// MemoryCache implements the userprefs.Cache interface using an in-memory map.
// It provides a thread-safe caching mechanism with support for item expiration and
// automatic garbage collection of expired items.
// All operations are protected by an internal sync.RWMutex.
type MemoryCache struct {
	mu     sync.RWMutex
	items  map[string]item
	stop   chan struct{} // Channel to signal gc goroutine to stop
	once   sync.Once     // Ensures stop channel is closed and items are cleared only once
	closed atomic.Bool   // Flag to indicate if the cache is closed (atomic for lock-free reads)
}

// NewMemoryCache initializes and returns a new MemoryCache instance.
// It also starts a background goroutine that periodically scans for and removes expired items
// from the cache. This goroutine is stopped when the Close method is called.
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]item),
		stop:  make(chan struct{}),
	}
	// closed is initialized to false by default (atomic.Bool zero value is false)
	go cache.gc()
	return cache
}

// Get retrieves an item from the memory cache by its key.
// The 'ctx' parameter is present for interface compliance but is not used in this implementation.
// It returns the cached item as a byte slice ([]byte) and nil error if the key is found and not expired.
// If the key does not exist or if the item has expired, it returns nil for the byte slice and
// userprefs.ErrNotFound from the parent package.
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	if c.isClosed() {
		return nil, userprefs.ErrCacheClosed
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, exists := c.items[key]
	if !exists {
		return nil, userprefs.ErrNotFound // Use error from main userprefs package
	}

	if !it.expiration.IsZero() && time.Now().After(it.expiration) {
		// For key expired, we could also use userprefs.ErrNotFound or a more specific one if available.
		// For now, let's assume manager treats any non-nil error other than userprefs.ErrNotFound from cache as a problem.
		// If key expired should be treated as a cache miss by the manager, this should also be userprefs.ErrNotFound.
		// Let's change it to userprefs.ErrNotFound for now to simplify manager logic, assuming expired = not found for manager's purposes.
		return nil, userprefs.ErrNotFound // Treat expired as not found for the manager
	}

	return it.value, nil
}

// Set adds an item (as a byte slice) to the memory cache with the given key, applying an optional TTL.
// The 'ctx' parameter is present for interface compliance but is not used in this implementation.
// The 'value' parameter must be a byte slice.
// If 'ttl' (time-to-live) is greater than zero, the item will be marked for expiration after that duration.
// If 'ttl' is zero or negative, the item will not expire (it will persist until explicitly deleted or the cache is closed).
// This method currently always returns nil error.
func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if c.isClosed() {
		return userprefs.ErrCacheClosed
	}

	// Defensive copy *before* we lock.
	copiedValue := make([]byte, len(value))
	copy(copiedValue, value)

	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}
	c.items[key] = item{value: copiedValue, expiration: expiration}
	return nil
}

// Delete removes an item from the memory cache by its key.
// The 'ctx' parameter is present for interface compliance but is not used in this implementation.
// This operation is idempotent: if the key does not exist, it does nothing and returns nil error.
// This method currently always returns nil error.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	if c.isClosed() {
		return userprefs.ErrCacheClosed
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Close stops the background garbage collection goroutine and clears all items from the memory cache.
// This method should be called when the MemoryCache is no longer needed to free resources.
// It effectively resets the cache to an empty state.
// This method currently always returns nil error.
func (c *MemoryCache) Close() error {
	c.once.Do(func() {
		c.mu.Lock()
		// Signal the gc goroutine to stop.
		// This is safe to do only once thanks to c.once.
		close(c.stop)

		// Clear items to reset the cache state.
		c.items = make(map[string]item)
		c.closed.Store(true) // Set the closed flag atomically
		c.mu.Unlock()
	})
	return nil
}

// isClosed checks if the cache has been closed.
func (c *MemoryCache) isClosed() bool {
	return c.closed.Load()
}

// gc runs a garbage collection process that periodically removes expired items.
func (c *MemoryCache) gc() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			for key, it := range c.items {
				if !it.expiration.IsZero() && time.Now().After(it.expiration) {
					delete(c.items, key)
				}
			}
			c.mu.Unlock()
		case <-c.stop:
			return // Exit goroutine
		}
	}
}
