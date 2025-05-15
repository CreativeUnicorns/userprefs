package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/redis/go-redis/v9"
)

// MockRedisClient is a mock implementation of redis.Client
type MockRedisClient struct {
	data map[string]string
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

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	_, _ = ctx.Deadline()
	_ = expiration.Abs().Milliseconds()
	if jsonVal, ok := value.([]byte); ok {
		m.data[key] = string(jsonVal)
	} else {
		m.data[key] = value.(string)
	}
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

func (m *MockRedisClient) Close() error {
	return nil
}

func TestRedisCache_GetSetDelete(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockRedisClient()
	redisCache := &RedisCache{client: mockClient}

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

	retrievedBytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Expected Get to return []byte, got %T", val)
	}

	// When value (which is []byte("test_value")) is set via RedisCache.Set,
	// it's first JSON marshalled. json.Marshal([]byte("test_value")) results in
	// the JSON string literal "dGVzdF92YWx1ZQ==" (base64 encoded content, quoted).
	// So, we expect retrievedBytes to be this JSON string literal.
	expectedMarshalledBytes, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Failed to marshal expected value for comparison: %v", err)
	}

	if !bytes.Equal(retrievedBytes, expectedMarshalledBytes) {
		t.Errorf("Retrieved value mismatch: got %s, want %s", string(retrievedBytes), string(expectedMarshalledBytes))
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
	redisCache := &RedisCache{client: mockClient}

	// Close cache
	if err := redisCache.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Attempt to set after closing (should not fail in mock)
	err := redisCache.Set(context.Background(), "key", "value", time.Minute)
	if err != nil {
		t.Errorf("Set after close failed: %v", err)
	}
}
