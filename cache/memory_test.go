package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CreativeUnicorns/userprefs"
)

func TestMemoryCache_GetSet(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	key := "testKey"
	value := "testValue"

	// Test Set
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	val, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != value {
		t.Errorf("Expected '%s', got '%v'", value, val)
	}

	// Test Get non-existent key
	_, err = cache.Get(ctx, "nonExistentKey")
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrNotFound for non-existent key, got: %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	key := "deleteKey"
	value := "deleteValue"

	// Set key
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete key
	if err := cache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Ensure key is deleted
	_, err := cache.Get(ctx, key)
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrNotFound for deleted key, got: %v", err)
	}
}

func TestMemoryCache_Close(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()

	// Set a key
	if err := cache.Set(ctx, "key", "value", time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Close the cache
	if err := cache.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Attempt to get a key after closing
	_, err := cache.Get(ctx, "key")
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrNotFound after closing, got: %v", err)
	}
}

func TestMemoryCache_GarbageCollection(t *testing.T) {
	ctx := context.Background()
	cache := NewMemoryCache()
	defer func() {
		err := cache.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	// Set multiple keys with different TTLs
	if err := cache.Set(ctx, "key1", "value1", time.Millisecond*100); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", "value2", time.Millisecond*200); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(ctx, "key3", "value3", 0); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait for garbage collector to run
	time.Sleep(time.Millisecond * 250)

	// key1 and key2 should be expired, key3 should exist
	_, err := cache.Get(ctx, "key1")
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrKeyExpired for key1, got: %v", err)
	}

	_, err = cache.Get(ctx, "key2")
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrKeyExpired for key2, got: %v", err)
	}

	val, err := cache.Get(ctx, "key3")
	if err != nil {
		t.Fatalf("Expected key3 to exist, got error: %v", err)
	}
	if val != "value3" {
		t.Errorf("Expected 'value3', got '%v'", val)
	}
}
