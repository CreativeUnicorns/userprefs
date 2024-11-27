package cache

import (
	"context"
	"testing"
	"time"
)

// TestCacheInterface ensures that MemoryCache and RedisCache implement the Cache interface
func TestCacheInterface(t *testing.T) {
	t.Name()
	var _ Cache = NewMemoryCache()

	// Since RedisCache requires a running Redis instance, we'll skip testing it here.
	// Implement mock Redis client and test RedisCache in a separate test file.
}

func TestCache_GetSetDelete_Close(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	// Test Set
	err := cache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got '%v'", val)
	}

	// Test Delete
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Ensure key is deleted
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Errorf("Expected error for deleted key, got none")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	// Set key with short TTL
	err := cache.Set(ctx, "tempKey", "tempValue", time.Millisecond*100)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Ensure key exists immediately
	val, err := cache.Get(ctx, "tempKey")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "tempValue" {
		t.Errorf("Expected 'tempValue', got '%v'", val)
	}

	// Wait for expiration
	time.Sleep(time.Millisecond * 200)

	// Attempt to get expired key
	_, err = cache.Get(ctx, "tempKey")
	if err == nil {
		t.Errorf("Expected error for expired key, got none")
	}
}

func TestMemoryCache_NoExpiration(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	// Set key with no expiration
	err := cache.Set(ctx, "permanentKey", "permanentValue", 0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait to ensure no expiration
	time.Sleep(time.Millisecond * 200)

	// Get key
	val, err := cache.Get(ctx, "permanentKey")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "permanentValue" {
		t.Errorf("Expected 'permanentValue', got '%v'", val)
	}
}
