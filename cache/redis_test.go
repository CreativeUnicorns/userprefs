package cache

// import (
// 	"context"
// 	"encoding/json"
// 	"errors"
// 	"testing"
// 	"time"

// 	"github.com/redis/go-redis/v9"
// )

// // MockRedisClient is a mock implementation of redis.Client
// type MockRedisClient struct {
// 	data map[string]string
// }

// func NewMockRedisClient() *MockRedisClient {
// 	return &MockRedisClient{
// 		data: make(map[string]string),
// 	}
// }

// func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
// 	_, _ = ctx.Deadline()
// 	val, exists := m.data[key]
// 	if !exists {
// 		return redis.NewStringResult("", redis.Nil)
// 	}
// 	return redis.NewStringResult(val, nil)
// }

// func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
// 	_, _ = ctx.Deadline()
// 	_ = expiration.Abs().Milliseconds()
// 	if jsonVal, ok := value.([]byte); ok {
// 		m.data[key] = string(jsonVal)
// 	} else {
// 		m.data[key] = value.(string)
// 	}
// 	return redis.NewStatusResult("OK", nil)
// }

// func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
// 	_, _ = ctx.Deadline()
// 	count := 0
// 	for _, key := range keys {
// 		if _, exists := m.data[key]; exists {
// 			delete(m.data, key)
// 			count++
// 		}
// 	}
// 	return redis.NewIntResult(int64(count), nil)
// }

// func (m *MockRedisClient) Close() error {
// 	return nil
// }

// func TestRedisCache_GetSetDelete(t *testing.T) {
// 	ctx := context.Background()
// 	mockClient := NewMockRedisClient()
// 	redisCache := &RedisCache{client: mockClient}

// 	key := "redisKey"
// 	value := map[string]interface{}{
// 		"name": "test",
// 		"age":  30,
// 	}

// 	// Set value
// 	if err := redisCache.Set(ctx, key, value, time.Minute); err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}

// 	// Get value
// 	val, err := redisCache.Get(ctx, key)
// 	if err != nil {
// 		t.Fatalf("Get failed: %v", err)
// 	}

// 	// Unmarshal to map
// 	var retrieved map[string]interface{}
// 	data, ok := val.([]byte)
// 	if !ok {
// 		t.Fatalf("Expected []byte, got %T", val)
// 	}
// 	if err := json.Unmarshal(data, &retrieved); err != nil {
// 		t.Fatalf("Unmarshal failed: %v", err)
// 	}

// 	if retrieved["name"] != "test" || int(retrieved["age"].(float64)) != 30 {
// 		t.Errorf("Retrieved value mismatch: %v", retrieved)
// 	}

// 	// Delete value
// 	if err := redisCache.Delete(ctx, key); err != nil {
// 		t.Fatalf("Delete failed: %v", err)
// 	}

// 	// Ensure deletion
// 	_, err = redisCache.Get(ctx, key)
// 	if !errors.Is(err, redis.Nil) {
// 		t.Errorf("Expected redis.Nil error, got: %v", err)
// 	}
// }

// func TestRedisCache_Close(t *testing.T) {
// 	mockClient := NewMockRedisClient()
// 	redisCache := &RedisCache{client: mockClient}

// 	// Close cache
// 	if err := redisCache.Close(); err != nil {
// 		t.Fatalf("Close failed: %v", err)
// 	}

// 	// Attempt to set after closing (should not fail in mock)
// 	err := redisCache.Set(context.Background(), "key", "value", time.Minute)
// 	if err != nil {
// 		t.Errorf("Set after close failed: %v", err)
// 	}
// }
