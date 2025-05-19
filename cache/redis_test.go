package cache

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a mock implementation of redis.Client
type MockRedisClient struct {
	data        map[string]string
	PingErr     error // Error to return on Ping
	CloseCalled bool  // True if Close was called
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
	}
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	_, _ = ctx.Deadline()
	val, exists := m.data[key]
	if !exists {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(val, nil)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value []byte, expiration time.Duration) *redis.StatusCmd {
	_, _ = ctx.Deadline()
	_ = expiration.Abs().Milliseconds()
	m.data[key] = string(value) // value is already []byte, store as string in mock
	return redis.NewStatusResult("OK", nil)
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	_, _ = ctx.Deadline()
	count := 0
	for _, key := range keys {
		if _, exists := m.data[key]; exists {
			delete(m.data, key)
			count++
		}
	}
	return redis.NewIntResult(int64(count), nil)
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	_, _ = ctx.Deadline()
	if m.PingErr != nil {
		return redis.NewStatusResult("", m.PingErr)
	}
	return redis.NewStatusResult("PONG", nil)
}

func (m *MockRedisClient) Close() error {
	m.CloseCalled = true
	return nil
}

func TestRedisCache_GetSetDelete(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockRedisClient()

	// Mock redisNewClientFunc to return our mock client
	originalNewClientFunc := redisNewClientFunc
	redisNewClientFunc = func(_ *redis.Options) RedisClient {
		return mockClient
	}
	defer func() { redisNewClientFunc = originalNewClientFunc }() // Restore original

	redisCache, err := NewRedisCache() // Use default options, client is mocked
	if err != nil {
		t.Fatalf("NewRedisCache failed: %v", err)
	}
	if redisCache == nil {
		t.Fatal("NewRedisCache returned nil cache")
	}

	key := "redisKey"
	value := []byte("test_value")

	// Set value
	if err := redisCache.Set(ctx, key, value, time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get value
	val, err := redisCache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	retrievedBytes := val // val is already []byte from redisCache.Get

	// The retrievedBytes should be identical to the original 'value' []byte passed to Set,
	// as RedisCache now directly stores/retrieves the []byte without further marshalling.
	if !bytes.Equal(retrievedBytes, value) {
		t.Errorf("Retrieved value mismatch: got %s, want %s", string(retrievedBytes), string(value))
	}

	// Delete value
	if err := redisCache.Delete(ctx, key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Ensure deletion
	_, err = redisCache.Get(ctx, key)
	if !errors.Is(err, userprefs.ErrNotFound) {
		t.Errorf("Expected ErrNotFound error, got: %v", err)
	}
}

func TestRedisCache_Close(t *testing.T) {
	mockClient := NewMockRedisClient()

	originalNewClientFunc := redisNewClientFunc
	redisNewClientFunc = func(_ *redis.Options) RedisClient {
		return mockClient
	}
	defer func() { redisNewClientFunc = originalNewClientFunc }()

	redisCache, err := NewRedisCache() // Use default options, client is mocked
	if err != nil {
		t.Fatalf("NewRedisCache failed: %v", err)
	}
	if redisCache == nil {
		t.Fatal("NewRedisCache returned nil cache")
	}

	// Close cache
	if err := redisCache.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Attempt to set after closing (should not fail in mock)
	err = redisCache.Set(context.Background(), "key", []byte("value"), time.Minute)
	if err != nil {
		t.Errorf("Set after close failed: %v", err)
	}

	if !mockClient.CloseCalled {
		t.Error("Expected client.Close to be called, but it wasn't")
	}
}

func TestNewRedisCache_PingFailure(t *testing.T) {
	mockClient := NewMockRedisClient()
	expectedPingErr := errors.New("simulated ping failure")
	mockClient.PingErr = expectedPingErr

	originalNewClientFunc := redisNewClientFunc
	redisNewClientFunc = func(_ *redis.Options) RedisClient {
		return mockClient
	}
	defer func() { redisNewClientFunc = originalNewClientFunc }()

	_, err := NewRedisCache(WithRedisAddress("localhost:1234")) // Address is just for error msg context
	if err == nil {
		t.Fatal("NewRedisCache did not return error on ping failure")
	}

	if !errors.Is(err, expectedPingErr) {
		t.Errorf("Expected error to wrap '%v', got '%v'", expectedPingErr, err)
	}
	if !mockClient.CloseCalled {
		t.Error("Expected client.Close to be called on ping failure, but it wasn't")
	}
}
