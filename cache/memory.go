// Package cache provides in-memory caching implementations.
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// item represents a single cache item with a value and an expiration time.
type item struct {
	value      interface{}
	expiration time.Time
}

// MemoryCache implements the Cache interface using an in-memory store.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

// NewMemoryCache initializes a new MemoryCache instance.
// It starts a garbage collection goroutine to clean expired items.
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]item),
	}
	go cache.gc()
	return cache
}

// Get retrieves a value from the memory cache by key.
// It returns an error if the key does not exist or has expired.
func (c *MemoryCache) Get(_ context.Context, key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	it, exists := c.items[key]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	if !it.expiration.IsZero() && time.Now().After(it.expiration) {
		return nil, fmt.Errorf("key expired")
	}

	return it.value, nil
}

// Set stores a value in the memory cache with an optional TTL.
// If TTL is greater than zero, the key will expire after the duration.
func (c *MemoryCache) Set(_ context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	c.items[key] = item{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Delete removes a key from the memory cache.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

// Close clears all items from the memory cache.
func (c *MemoryCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]item)
	return nil
}

// gc runs a garbage collection process that periodically removes expired items.
func (c *MemoryCache) gc() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for key, it := range c.items {
			if !it.expiration.IsZero() && time.Now().After(it.expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
