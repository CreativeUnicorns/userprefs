// Package cache provides Redis-based caching implementations.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/redis/go-redis/v9"
)

// redisClientAdapter wraps a *redis.Client to satisfy the RedisClient interface,
// specifically ensuring the Set method signature matches.
type redisClientAdapter struct {
	*redis.Client
}

// Set on redisClientAdapter calls the underlying client's Set method.
// It takes []byte for value to match the RedisClient interface.
func (a *redisClientAdapter) Set(ctx context.Context, key string, value []byte, expiration time.Duration) *redis.StatusCmd {
	return a.Client.Set(ctx, key, value, expiration)
}

var (
	// redisNewClientFunc is a mockable function to create a RedisClient.
	// It defaults to creating an adapter around a standard redis.Client.
	redisNewClientFunc = func(opt *redis.Options) RedisClient {
		return &redisClientAdapter{Client: redis.NewClient(opt)}
	}
)

// RedisConfig holds configuration options for the Redis cache backend.
// Fields correspond to options available in github.com/redis/go-redis/v9.
// If a field is not set via a WithRedis... option, the go-redis library's default will be used.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int

	// Connection Pool Options
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Retry Options
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

// RedisOption defines a function signature for configuring RedisCache settings.
// These options are applied to a RedisConfig struct during the initialization of a RedisCache instance.
type RedisOption func(*RedisConfig)

// WithRedisAddress sets the Redis server address (e.g., "localhost:6379").
// If not specified, NewRedisCache defaults to "localhost:6379".
func WithRedisAddress(addr string) RedisOption {
	return func(c *RedisConfig) {
		c.Addr = addr
	}
}

// WithRedisPassword sets the password for Redis server authentication.
// Default is no password.
func WithRedisPassword(password string) RedisOption {
	return func(c *RedisConfig) {
		c.Password = password
	}
}

// WithRedisDB sets the Redis database number to select after connecting.
// Default is DB 0.
func WithRedisDB(db int) RedisOption {
	return func(c *RedisConfig) {
		c.DB = db
	}
}

// WithRedisPoolSize sets the maximum number of socket connections in the connection pool.
// Default is typically 10 connections per CPU core.
func WithRedisPoolSize(size int) RedisOption {
	return func(c *RedisConfig) {
		c.PoolSize = size
	}
}

// WithRedisMinIdleConns sets the minimum number of idle connections maintained in the pool.
// Default is 0 (no minimum).
func WithRedisMinIdleConns(conns int) RedisOption {
	return func(c *RedisConfig) {
		c.MinIdleConns = conns
	}
}

// WithRedisPoolTimeout sets the amount of time the client waits for a connection if all
// connections in the pool are busy before returning an error.
// Default is ReadTimeout + 1 second.
func WithRedisPoolTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.PoolTimeout = timeout
	}
}

// WithRedisDialTimeout sets the timeout for establishing new connections to the Redis server.
// Default is 5 seconds.
func WithRedisDialTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.DialTimeout = timeout
	}
}

// WithRedisReadTimeout sets the timeout for read operations on a Redis connection.
// Default is 3 seconds. Set to -1 for no timeout.
func WithRedisReadTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.ReadTimeout = timeout
	}
}

// WithRedisWriteTimeout sets the timeout for write operations on a Redis connection.
// Default is ReadTimeout. Set to -1 for no timeout.
func WithRedisWriteTimeout(timeout time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.WriteTimeout = timeout
	}
}

// WithRedisMaxRetries sets the maximum number of retries for a command before giving up.
// Default is 0 (no retries). Set to -1 for infinite retries.
func WithRedisMaxRetries(retries int) RedisOption {
	return func(c *RedisConfig) {
		c.MaxRetries = retries
	}
}

// WithRedisMinRetryBackoff sets the minimum backoff duration between command retries.
// Default is 8 milliseconds. Use -1 to disable backoff.
func WithRedisMinRetryBackoff(backoff time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.MinRetryBackoff = backoff
	}
}

// WithRedisMaxRetryBackoff sets the maximum backoff duration between command retries.
// Default is 512 milliseconds. Use -1 to disable backoff.
func WithRedisMaxRetryBackoff(backoff time.Duration) RedisOption {
	return func(c *RedisConfig) {
		c.MaxRetryBackoff = backoff
	}
}

// RedisClient defines an interface abstracting the methods used from `redis.Client`.
// This abstraction allows for easier mocking and testing of RedisCache.
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
	Ping(ctx context.Context) *redis.StatusCmd // Add Ping for connection check
}

// RedisCache implements the userprefs.Cache interface using a Redis backend.
// It leverages the github.com/redis/go-redis/v9 library for Redis communication.
type RedisCache struct {
	client RedisClient
}

// NewRedisCache creates a new RedisCache instance.
// It takes a list of RedisOption functions to configure the Redis client.
// Required options: WithRedisAddress.
// Optional options: WithRedisPassword, WithRedisDB, WithRedisPoolSize, etc.
//
// The function initializes a Redis client and pings it to ensure connectivity.
// The DialTimeout setting in RedisConfig (or its default) governs the timeout for the initial connection attempt during client creation.
// A separate context with a timeout (derived from DialTimeout or a default of 5s) is used for the initial Ping.
// Returns an error if client creation or the initial Ping fails.
func NewRedisCache(opts ...RedisOption) (*RedisCache, error) {
	cfg := RedisConfig{
		Addr: "localhost:6379", // Default address
		DB:   0,                // Default DB
		// Other fields will use go-redis defaults if not set by options
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	clientOpts := &redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		PoolTimeout:     cfg.PoolTimeout,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}

	client := redisNewClientFunc(clientOpts) // Use the mockable function

	// Use a timeout for the initial Ping
	pingCtxTimeout := 5 * time.Second
	if cfg.DialTimeout > 0 {
		// If a dial timeout is set, use it for ping context, but cap at a reasonable max for initial check.
		if cfg.DialTimeout < pingCtxTimeout {
			pingCtxTimeout = cfg.DialTimeout
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingCtxTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		// Log or handle client creation error
		if client != nil {
			_ = client.Close() // Attempt to close the client if ping fails
		}
		return nil, err
	}

	return &RedisCache{
		client: client,
	}, nil
}

// Get retrieves an item from Redis by its key.
// If the key is not found in Redis (redis.Nil error), it returns (nil, userprefs.ErrNotFound).
// For other Redis errors, it returns (nil, error_details).
// On success, it returns the item as a byte slice ([]byte) and nil error. The caller is responsible for unmarshalling this data.
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes() // Ensure .Bytes() is used
	if err == redis.Nil {
		return nil, userprefs.ErrNotFound // Cache miss, use error from main userprefs package
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from redis: %w", err) // Other Redis error
	}
	return data, nil // Return the []byte
}

// Set stores an item (as a byte slice) in Redis.
// The 'value' parameter must be a byte slice, typically pre-marshalled by the caller (e.g., the Manager).
// If 'ttl' (time-to-live) is greater than zero, the item will expire in Redis after that duration.
// If 'ttl' is zero or negative, the item will be stored without an expiration.
// Returns an error if the Redis SET operation fails.
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	expiration := ttl
	if ttl <= 0 {
		expiration = 0 // For Redis, 0 means no expiration.
	}

	if err := c.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set in redis: %w", err)
	}

	return nil
}

// Delete removes an item from Redis by its key.
// The underlying Redis DEL command is idempotent. This method returns an error only if the DEL command itself fails.
// If the key does not exist, Redis DEL command still returns success (0 keys deleted), so no error is returned by this method in that case.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from redis: %w", err)
	}

	return nil
}

// Close closes the underlying Redis client connection pool.
// It should be called when the RedisCache is no longer needed to release resources.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
