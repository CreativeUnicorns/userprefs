package cache

import (
	"bytes"
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
	value := []byte("testValue")

	// Test Set
	if err := cache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	val, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !bytes.Equal(val, value) {
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
	value := []byte("deleteValue")

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

	t.Run("idempotent_close", func(t *testing.T) {
		cache := NewMemoryCache()
		// Call Close multiple times
		if err := cache.Close(); err != nil {
			t.Fatalf("First Close() failed: %v", err)
		}
		if err := cache.Close(); err != nil {
			t.Fatalf("Second Close() failed (idempotency check): %v", err)
		}
	})

	t.Run("operations_after_close", func(t *testing.T) {
		cache := NewMemoryCache()
		// Set a key before closing
		if err := cache.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
			t.Fatalf("Set before close failed: %v", err)
		}

		// Close the cache
		if err := cache.Close(); err != nil {
			t.Fatalf("Close() failed: %v", err)
		}

		// Attempt Get after closing
		_, errGet := cache.Get(ctx, "key")
		if !errors.Is(errGet, userprefs.ErrCacheClosed) {
			t.Errorf("Expected ErrCacheClosed for Get after closing, got: %v", errGet)
		}

		// Attempt Set after closing
		errSet := cache.Set(ctx, "anotherKey", []byte("anotherValue"), time.Minute)
		if !errors.Is(errSet, userprefs.ErrCacheClosed) {
			t.Errorf("Expected ErrCacheClosed for Set after closing, got: %v", errSet)
		}

		// Attempt Delete after closing
		errDel := cache.Delete(ctx, "key")
		if !errors.Is(errDel, userprefs.ErrCacheClosed) {
			t.Errorf("Expected ErrCacheClosed for Delete after closing, got: %v", errDel)
		}
	})
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
	if err := cache.Set(ctx, "key1", []byte("value1"), time.Millisecond*100); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(ctx, "key2", []byte("value2"), time.Millisecond*200); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if err := cache.Set(ctx, "key3", []byte("value3"), 0); err != nil {
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
	if !bytes.Equal(val, []byte("value3")) {
		t.Errorf("Expected 'value3', got '%v'", val)
	}
}
