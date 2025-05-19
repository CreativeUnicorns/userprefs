// Package storage provides a PostgreSQL-based implementation of the Storage interface.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/CreativeUnicorns/userprefs"
)

// sqlOpenFunc is a package-level variable that can be overridden for testing.
var sqlOpenFunc = sql.Open

// PostgresConfig holds configuration options for the PostgreSQL storage backend.
type PostgresConfig struct {
	// DSN is the full Data Source Name (e.g., "postgres://user:password@host:port/dbname?sslmode=disable").
	// If provided, DSN takes precedence over individual connection parameters like Host, Port, User, etc.
	DSN string
	// Host is the server hostname or IP address.
	Host string
	// Port is the server port number.
	Port int
	// User is the database username.
	User string
	// Password is the database password.
	Password string
	// DBName is the name of the database to connect to.
	DBName string
	// SSLMode specifies the SSL/TLS security mode (e.g., "disable", "require", "verify-full").
	SSLMode string
	// ConnectTimeout is the maximum time to wait for a connection to be established.
	// A zero value means no timeout.
	ConnectTimeout time.Duration

	// Connection Pool settings
	// MaxOpenConns is the maximum number of open connections to the database.
	// If MaxOpenConns is 0, then there is no limit on the number of open connections.
	// The default is 0 (unlimited).
	MaxOpenConns int
	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	// If MaxIdleConns is 0, no idle connections are retained.
	// The default is 2. This value should be less than or equal to MaxOpenConns.
	MaxIdleConns int
	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Expired connections may be closed lazily before reuse.
	// If d <= 0, connections are not closed due to a connection's age.
	ConnMaxLifetime time.Duration
	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Expired connections may be closed lazily before reuse.
	// If d <= 0, connections are not closed due to a connection's idle time.
	ConnMaxIdleTime time.Duration
}

// PostgresOption is a function that configures PostgresStorage.
type PostgresOption func(*PostgresConfig)

// WithPostgresDSN sets the full DSN string.
// If provided, this will override individual connection parameters like Host, Port, etc.
func WithPostgresDSN(dsn string) PostgresOption {
	return func(c *PostgresConfig) {
		c.DSN = dsn
	}
}

// WithPostgresHost sets the database host.
func WithPostgresHost(host string) PostgresOption {
	return func(c *PostgresConfig) {
		c.Host = host
	}
}

// WithPostgresPort sets the database port.
func WithPostgresPort(port int) PostgresOption {
	return func(c *PostgresConfig) {
		c.Port = port
	}
}

// WithPostgresUser sets the database user.
func WithPostgresUser(user string) PostgresOption {
	return func(c *PostgresConfig) {
		c.User = user
	}
}

// WithPostgresPassword sets the database password.
func WithPostgresPassword(password string) PostgresOption {
	return func(c *PostgresConfig) {
		c.Password = password
	}
}

// WithPostgresDBName sets the database name.
func WithPostgresDBName(dbname string) PostgresOption {
	return func(c *PostgresConfig) {
		c.DBName = dbname
	}
}

// WithPostgresSSLMode sets the SSL mode (e.g., "disable", "require", "verify-full").
func WithPostgresSSLMode(sslmode string) PostgresOption {
	return func(c *PostgresConfig) {
		c.SSLMode = sslmode
	}
}

// WithPostgresConnectTimeout sets the connection timeout.
func WithPostgresConnectTimeout(timeout time.Duration) PostgresOption {
	return func(c *PostgresConfig) {
		c.ConnectTimeout = timeout
	}
}

// WithPostgresMaxOpenConns sets the maximum number of open connections to the database.
func WithPostgresMaxOpenConns(n int) PostgresOption {
	return func(c *PostgresConfig) {
		c.MaxOpenConns = n
	}
}

// WithPostgresMaxIdleConns sets the maximum number of connections in the idle connection pool.
func WithPostgresMaxIdleConns(n int) PostgresOption {
	return func(c *PostgresConfig) {
		c.MaxIdleConns = n
	}
}

// WithPostgresConnMaxLifetime sets the maximum amount of time a connection may be reused.
func WithPostgresConnMaxLifetime(d time.Duration) PostgresOption {
	return func(c *PostgresConfig) {
		c.ConnMaxLifetime = d
	}
}

// WithPostgresConnMaxIdleTime sets the maximum amount of time a connection may be idle before being closed.
func WithPostgresConnMaxIdleTime(d time.Duration) PostgresOption {
	return func(c *PostgresConfig) {
		c.ConnMaxIdleTime = d
	}
}

const (
	createTableSQL = `
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value JSONB NOT NULL,
			default_value JSONB,
			type TEXT NOT NULL,
			category TEXT,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, key)
		);
		
		CREATE INDEX IF NOT EXISTS idx_user_preferences_category 
		ON user_preferences(user_id, category);
	`

	insertSQL = `
		INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, key) 
		DO UPDATE SET value = $3, default_value = $4, updated_at = $7
	`

	selectSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND key = $2
	`

	selectByCategorySQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND category = $2
	`

	selectAllSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1
	`

	deleteSQL = `
		DELETE FROM user_preferences 
		WHERE user_id = $1 AND key = $2
	`
)

// PostgresStorage implements the Storage interface using PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage initializes a new PostgresStorage instance, configured by the provided options.
// It establishes a connection to the PostgreSQL database, applies connection pool settings,
// pings the database to verify connectivity, and runs necessary database migrations.
//
// Returns a pointer to an initialized PostgresStorage and nil error on success.
// Returns nil and an error if configuration is invalid, connection fails, ping fails,
// or migrations fail. Errors from underlying database operations are wrapped.
func NewPostgresStorage(opts ...PostgresOption) (*PostgresStorage, error) {
	cfg := PostgresConfig{
		Host:    "localhost",
		Port:    5432,
		User:    "postgres", // Common default, adjust if necessary
		SSLMode: "disable",  // Default to disable for local dev, encourage override for prod
		// DBName is intentionally left blank; it's usually required.
		// Password is intentionally left blank.
		// ConnectTimeout: 0 (driver default)
		// MaxIdleConns: 2 (driver default)
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	dsn := cfg.DSN
	if dsn == "" {
		var params []string
		if cfg.Host != "" {
			params = append(params, "host="+cfg.Host)
		}
		if cfg.Port > 0 {
			params = append(params, "port="+strconv.Itoa(cfg.Port))
		}
		if cfg.User != "" {
			params = append(params, "user="+cfg.User)
		}
		if cfg.Password != "" {
			params = append(params, "password="+cfg.Password)
		}
		if cfg.DBName != "" {
			params = append(params, "dbname="+cfg.DBName)
		} else {
			return nil, fmt.Errorf("postgres: DBName must be configured")
		}
		if cfg.SSLMode != "" {
			params = append(params, "sslmode="+cfg.SSLMode)
		}
		if cfg.ConnectTimeout > 0 {
			params = append(params, "connect_timeout="+strconv.Itoa(int(cfg.ConnectTimeout.Seconds())))
		}
		dsn = strings.Join(params, " ")
	}

	db, err := sqlOpenFunc("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to open database connection: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	// Use a reasonable default timeout if none is provided
	timeout := cfg.ConnectTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second // Default to 5 seconds if no timeout is specified
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to ping database", "error", err)
		if db != nil {
			_ = db.Close() // Attempt to close if ping fails
		}
		return nil, fmt.Errorf("postgres: failed to ping database: %w", err)
	}

	storage := &PostgresStorage{db: db}
	if err := storage.migrate(); err != nil {
		slog.ErrorContext(ctx, "Failed to apply migrations", "error", err)
		if db != nil {
			_ = db.Close() // Attempt to close if migration fails
		}
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs the necessary database migrations.
func (s *PostgresStorage) migrate() error {
	_, err := s.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("postgres: failed to execute create table statement: %w", err)
	}
	return nil
}

// Get retrieves a specific preference for a given user ID and key.
// The provided context.Context can be used for cancellation or timeouts.
//
// On success, it returns a pointer to the userprefs.Preference and a nil error.
// If the preference is not found, it returns nil and userprefs.ErrNotFound.
// If there's an issue with database interaction (e.g., connection problem, query failure),
// a wrapped error is returned.
// If the stored preference value or default value cannot be unmarshalled from JSON,
// it returns nil and an error wrapping userprefs.ErrSerialization.
func (s *PostgresStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON []byte
	var defaultValueJSON []byte // Added for DefaultValue
	var category sql.NullString // Use sql.NullString for nullable category

	err := s.db.QueryRowContext(ctx, selectSQL, userID, key).Scan(
		&pref.UserID,
		&pref.Key,
		&valueJSON,
		&defaultValueJSON, // Added for DefaultValue
		&pref.Type,
		&category, // Scan into sql.NullString
		&pref.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, userprefs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to scan preference for user '%s', key '%s': %w", userID, key, err)
	}

	if category.Valid {
		pref.Category = category.String
	} else {
		pref.Category = "" // Or handle as appropriate for your application logic
	}

	if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
		return nil, fmt.Errorf("%w: postgres: failed to unmarshal value for user '%s', key '%s': %v", userprefs.ErrSerialization, userID, key, err)
	}

	// Unmarshal DefaultValue, allow it to be null
	if defaultValueJSON != nil && string(defaultValueJSON) != "null" {
		if err := json.Unmarshal(defaultValueJSON, &pref.DefaultValue); err != nil {
			return nil, fmt.Errorf("%w: postgres: failed to unmarshal default_value for user '%s', key '%s': %v", userprefs.ErrSerialization, userID, key, err)
		}
	} else {
		pref.DefaultValue = nil // Ensure it's nil if DB value is NULL or empty
	}

	return &pref, nil
}

// Set stores or updates a user's preference. The provided context.Context can be used
// for cancellation or timeouts.
// The pref.Value and pref.DefaultValue fields are marshalled to JSONB for storage.
// This operation is an "upsert": if a preference with the given userID and key
// already exists, it is updated; otherwise, a new preference is created.
// The UpdatedAt field of the preference is set to the current time by the database.
//
// Returns nil on successful creation or update.
// Returns an error if marshalling to JSON fails (wrapping userprefs.ErrSerialization),
// or if the database operation fails (wrapped error).
func (s *PostgresStorage) Set(ctx context.Context, pref *userprefs.Preference) error {
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return fmt.Errorf("%w: postgres: failed to marshal value for key '%s': %v", userprefs.ErrSerialization, pref.Key, err)
	}

	defaultValueJSON, err := json.Marshal(pref.DefaultValue)
	if err != nil {
		// Handle nil DefaultValue gracefully: if it's nil, marshal it as SQL NULL / JSON null
		if pref.DefaultValue == nil {
			defaultValueJSON = []byte("null")
		} else {
			return fmt.Errorf("%w: postgres: failed to marshal default_value for key '%s': %v", userprefs.ErrSerialization, pref.Key, err)
		}
	}

	_, err = s.db.ExecContext(ctx, insertSQL,
		pref.UserID,
		pref.Key,
		valueJSON,
		defaultValueJSON,
		pref.Type,
		pref.Category,
		pref.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("postgres: failed to execute insert/update for user '%s', key '%s': %w", pref.UserID, pref.Key, err)
	}

	return nil
}

// GetByCategory retrieves all preferences for a given user ID that belong to the specified category.
// The provided context.Context can be used for cancellation or timeouts.
//
// On success, it returns a map where keys are preference keys and values are pointers
// to userprefs.Preference, and a nil error.
// If no preferences are found for the category, it returns an empty map and a nil error.
// If there's an issue with database interaction, a wrapped error is returned.
// If any stored preference value or default value cannot be unmarshalled from JSON,
// it returns nil and an error wrapping userprefs.ErrSerialization.
func (s *PostgresStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to query preferences by category for user '%s', category '%s': %w", userID, category, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(ctx, rows)
}

// GetAll retrieves all preferences for a given user ID.
// The provided context.Context can be used for cancellation or timeouts.
//
// On success, it returns a map where keys are preference keys and values are pointers
// to userprefs.Preference, and a nil error.
// If the user has no preferences, it returns an empty map and a nil error.
// If there's an issue with database interaction, a wrapped error is returned.
// If any stored preference value or default value cannot be unmarshalled from JSON,
// it returns nil and an error wrapping userprefs.ErrSerialization.
func (s *PostgresStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to query all preferences for user '%s': %w", userID, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(ctx, rows)
}

// Delete removes a specific preference for a given user ID and key.
// The provided context.Context can be used for cancellation or timeouts.
//
// Returns nil on successful deletion.
// If the preference to be deleted is not found, it returns userprefs.ErrNotFound.
// If there's an issue with the database operation, a wrapped error is returned.
func (s *PostgresStorage) Delete(ctx context.Context, userID, key string) error {
	result, err := s.db.ExecContext(ctx, deleteSQL, userID, key)
	if err != nil {
		return fmt.Errorf("postgres: failed to execute delete for user '%s', key '%s': %w", userID, key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: failed to get affected rows for delete user '%s', key '%s': %w", userID, key, err)
	}

	if rowsAffected == 0 {
		return userprefs.ErrNotFound // No rows were deleted, so the preference was not found.
	}

	return nil
}

// Close closes the underlying PostgreSQL database connection pool.
// It is important to call Close when the PostgresStorage is no longer needed
// to release database resources.
//
// Returns an error if closing the connection pool fails.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
// It handles closing the rows object, returning any error from rows.Close()
// if no other error occurred, or wrapping it if another error was primary.
func (s *PostgresStorage) scanPreferences(ctx context.Context, rows *sql.Rows) (prefsMap map[string]*userprefs.Preference, err error) {
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil {
			if err == nil { // If no primary error, the closeErr becomes the primary error
				err = fmt.Errorf("postgres: failed to close rows: %w", closeErr)
			} else {
				// If there was already an error, wrap the original error with closeErr info.
				err = fmt.Errorf("postgres: primary error: %w; additionally failed to close rows: %v", err, closeErr)
			}
		}
	}()

	prefsMap = make(map[string]*userprefs.Preference)

	for rows.Next() {
		// Check for context cancellation before processing each row
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return // prefsMap will be nil, err will be set by defer or here
		default:
			// Continue processing
		}

		var pref userprefs.Preference
		var valueJSON []byte
		var defaultValueJSON []byte // Stored as JSONB, can be null
		var category sql.NullString // Use sql.NullString for nullable category

		scanErr := rows.Scan(
			&pref.UserID,
			&pref.Key,
			&valueJSON,
			&defaultValueJSON,
			&pref.Type,
			&category, // Scan into sql.NullString
			&pref.UpdatedAt,
		)
		if scanErr != nil {
			err = fmt.Errorf("postgres: failed to scan preference row: %w", scanErr)
			return
		}

		if category.Valid {
			pref.Category = category.String
		} else {
			pref.Category = "" // Or handle as appropriate for your application logic
		}

		if unmarshalErr := json.Unmarshal(valueJSON, &pref.Value); unmarshalErr != nil {
			err = fmt.Errorf("%w: postgres: failed to unmarshal value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, unmarshalErr)
			return
		}

		// Unmarshal DefaultValue, allow it to be null (represented by nil []byte or JSON 'null')
		if defaultValueJSON != nil && string(defaultValueJSON) != "null" {
			if unmarshalErr := json.Unmarshal(defaultValueJSON, &pref.DefaultValue); unmarshalErr != nil {
				err = fmt.Errorf("%w: postgres: failed to unmarshal default_value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, unmarshalErr)
				return
			}
		} else {
			pref.DefaultValue = nil
		}

		prefsMap[pref.Key] = &pref
	}

	// Check for errors encountered during iteration.
	if iterationErr := rows.Err(); iterationErr != nil {
		err = fmt.Errorf("postgres: error iterating preference rows: %w", iterationErr)
		return
	}

	return // prefsMap will be returned, err is nil or set by defer
}
