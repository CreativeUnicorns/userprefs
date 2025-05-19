// Package storage provides a SQLite-based implementation of the Storage interface.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings" // Added for strings.Contains
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/CreativeUnicorns/userprefs"
)

// sqliteOpenFuncType defines the signature for a function that opens a database connection.
// This is used to allow mocking of sql.Open in tests.
type sqliteOpenFuncType func(driverName, dataSourceName string) (*sql.DB, error)

// sqliteOpen is a package-level variable that holds the function used to open DB connections.
// It defaults to sql.Open but can be overridden in tests.
var sqliteOpen sqliteOpenFuncType = sql.Open

const (
	sqliteCreateTableSQL = `
		CREATE TABLE IF NOT EXISTS user_preferences (
			user_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			default_value TEXT,
			type TEXT NOT NULL,
			category TEXT,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, key)
		);
		
		CREATE INDEX IF NOT EXISTS idx_user_preferences_category 
		ON user_preferences(user_id, category);
	`

	sqliteInsertSQL = `
		INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, key) 
		DO UPDATE SET value = ?, default_value = ?, updated_at = ?
	`

	sqliteSelectSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = ? AND key = ?
	`

	sqliteSelectByCategorySQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = ? AND category = ?
	`

	sqliteSelectAllSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = ?
	`

	sqliteDeleteSQL = `
		DELETE FROM user_preferences 
		WHERE user_id = ? AND key = ?
	`
)

// SQLiteConfig holds configuration options for the SQLite storage backend.
type SQLiteConfig struct {
	// FilePath specifies the path to the SQLite database file.
	// If set to ":memory:", an in-memory database is used.
	// This field is typically set by the NewSQLiteStorage function's first argument.
	FilePath string
	// InMemory indicates if the database should be purely in-memory.
	// This is true if FilePath is ":memory:".
	InMemory bool
	// WALMode enables or disables SQLite's Write-Ahead Logging (WAL) mode.
	// WAL can improve concurrency and performance. Enabled by default.
	WALMode bool
	// BusyTimeout sets the SQLite `busy_timeout` pragma, which specifies how long
	// a connection should wait for a lock to be released before returning SQLITE_BUSY.
	// Default is 5 seconds.
	BusyTimeout time.Duration
	// JournalMode sets the SQLite `journal_mode` pragma (e.g., "DELETE", "WAL", "MEMORY").
	// If WALMode is true (default), JournalMode will effectively be "WAL".
	// This option is primarily for explicitly setting a non-WAL journal mode when WALMode is false.
	JournalMode string // Moved JournalMode field here
	// ExtraParams allows specifying additional DSN parameters for SQLite.
	// Keys are parameter names (e.g., "_cache_size"), values are their string representations.
	ExtraParams map[string]string
} // End of SQLiteConfig struct

// SQLiteOption is a function type for configuring SQLiteStorage.
// It allows for a flexible way to set options on the SQLiteConfig struct.
type SQLiteOption func(*SQLiteConfig)

// WithSQLiteWAL enables or disables SQLite\'s Write-Ahead Logging (WAL) mode.
// WAL can improve concurrency and performance.
func WithSQLiteWAL(enable bool) SQLiteOption {
	return func(c *SQLiteConfig) {
		c.WALMode = enable
	}
}

// WithSQLiteBusyTimeout sets the SQLite `busy_timeout` pragma.
// This specifies how long a connection should wait for a lock to be released
// before returning SQLITE_BUSY. The duration `d` is converted to milliseconds.
func WithSQLiteBusyTimeout(d time.Duration) SQLiteOption {
	return func(c *SQLiteConfig) {
		c.BusyTimeout = d
	}
}

// WithSQLiteJournalMode sets the SQLite `journal_mode` pragma (e.g., "DELETE", "WAL", "MEMORY").
// If WALMode is true (default behavior, or set by WithSQLiteWAL(true)),
// JournalMode will effectively be "WAL" regardless of this setting unless WAL is later disabled.
// This option is primarily for explicitly setting a non-WAL journal mode when WALMode is false.
func WithSQLiteJournalMode(mode string) SQLiteOption {
	return func(c *SQLiteConfig) {
		c.JournalMode = mode
	}
}

// WithSQLiteExtraParam adds a custom DSN parameter for SQLite.
// This allows for fine-tuning SQLite behavior by setting less common pragmas or parameters
// directly in the DSN string. For example, `WithSQLiteExtraParam("_cache_size", "-2000")`
// would add `_cache_size=-2000` to the DSN, instructing SQLite to use up to 2MB (2000 KiB)
// of memory for its page cache. Refer to the official SQLite documentation for a list
// of available DSN parameters and pragmas.
func WithSQLiteExtraParam(key, value string) SQLiteOption {
	return func(c *SQLiteConfig) {
		if c.ExtraParams == nil {
			c.ExtraParams = make(map[string]string)
		}
		c.ExtraParams[key] = value
	}
}

// SQLiteStorage implements the Storage interface using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage initializes a new SQLiteStorage instance, configured by the filePath and provided options.
//
// The `filePath` argument specifies the database location:
// - Use a file path (e.g., "./userprefs.db") for a persistent disk-based database.
// - Use ":memory:" for a volatile in-memory database, useful for testing or temporary storage.
//
// Functional options (`opts`) allow customization of WAL mode, busy timeout, journal mode,
// and other DSN parameters. See `WithSQLiteWAL`, `WithSQLiteBusyTimeout`, etc.
//
// The function performs the following steps:
// 1. Applies configuration options.
// 2. Constructs the DSN string, including any extra parameters.
// 3. Opens a connection to the SQLite database using the DSN.
// 4. Pings the database to verify connectivity.
// 5. Applies database schema migrations (`sqliteCreateTableSQL`).
//
// Returns a pointer to an initialized SQLiteStorage and nil error on success.
// Returns nil and an error if configuration is invalid, DSN construction fails, connection fails,
// ping fails, or migrations fail. Errors from underlying database operations are wrapped.
func NewSQLiteStorage(filePath string, opts ...SQLiteOption) (*SQLiteStorage, error) {
	if filePath == "" {
		return nil, fmt.Errorf("sqlite: filePath cannot be empty")
	}

	cfg := SQLiteConfig{
		FilePath:    filePath,
		InMemory:    filePath == ":memory:",
		WALMode:     true, // Default WAL to true
		BusyTimeout: 5 * time.Second,
		ExtraParams: make(map[string]string),
		// JournalMode default is empty; SQLite handles it or WAL sets it.
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	dsnParams := url.Values{}

	// Explicitly set journal_mode based on WALMode or specified JournalMode.
	if cfg.WALMode {
		dsnParams.Set("_journal_mode", "WAL")
	} else if cfg.JournalMode != "" {
		dsnParams.Set("_journal_mode", cfg.JournalMode)
	}

	if cfg.BusyTimeout > 0 {
		dsnParams.Set("_busy_timeout", strconv.FormatInt(cfg.BusyTimeout.Milliseconds(), 10))
	}

	for k, v := range cfg.ExtraParams {
		dsnParams.Set(k, v)
	}

	dsn := cfg.FilePath
	if query := dsnParams.Encode(); query != "" {
		if strings.Contains(dsn, "?") {
			dsn += "&" + query
		} else {
			dsn += "?" + query
		}
	}

	db, err := sqliteOpen("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to open database with DSN '%s': %w", dsn, err)
	}

	ctx := context.Background() // Create a background context for the Ping operation
	err = db.PingContext(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to ping database", "error", err)
		if db != nil {
			_ = db.Close() // Attempt to close if ping fails
		}
		return nil, fmt.Errorf("sqlite: failed to ping database: %w", err)
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.migrate(); err != nil {
		slog.ErrorContext(ctx, "Failed to apply migrations", "error", err)
		if db != nil {
			_ = db.Close() // Attempt to close if migration fails
		}
		return nil, fmt.Errorf("sqlite: failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs the necessary database migrations.
func (s *SQLiteStorage) migrate() error {
	_, err := s.db.Exec(sqliteCreateTableSQL)
	if err != nil {
		return fmt.Errorf("sqlite: failed to execute create table statement: %w", err)
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
func (s *SQLiteStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON string
	var defaultValueJSON sql.NullString // Added for DefaultValue
	var category sql.NullString         // Use sql.NullString for nullable category

	err := s.db.QueryRowContext(ctx, sqliteSelectSQL, userID, key).Scan(
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
		return nil, fmt.Errorf("sqlite: failed to scan preference for user '%s', key '%s': %w", userID, key, err)
	}

	if category.Valid {
		pref.Category = category.String
	} else {
		pref.Category = "" // Or handle as appropriate for your application logic
	}

	if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
		return nil, fmt.Errorf("%w: sqlite: failed to unmarshal value for user '%s', key '%s': %v", userprefs.ErrSerialization, userID, key, err)
	}

	// Unmarshal DefaultValue, allow it to be null
	if defaultValueJSON.Valid && defaultValueJSON.String != "null" {
		if err := json.Unmarshal([]byte(defaultValueJSON.String), &pref.DefaultValue); err != nil {
			return nil, fmt.Errorf("%w: sqlite: failed to unmarshal default_value for user '%s', key '%s': %v", userprefs.ErrSerialization, userID, key, err)
		}
	} else {
		pref.DefaultValue = nil // Ensure it's nil if DB value is NULL or empty
	}

	return &pref, nil
}

// Set stores or updates a user's preference. The provided context.Context can be used
// for cancellation or timeouts.
// The pref.Value and pref.DefaultValue fields are marshalled to JSON (as TEXT) for storage.
// This operation is an "upsert" (INSERT ON CONFLICT DO UPDATE): if a preference with the
// given userID and key already exists, it is updated; otherwise, a new preference is created.
// The UpdatedAt field of the preference is set to the current time by the database.
//
// Returns nil on successful creation or update.
// Returns an error if marshalling to JSON fails (wrapping userprefs.ErrSerialization),
// or if the database operation fails (wrapped error).
func (s *SQLiteStorage) Set(ctx context.Context, pref *userprefs.Preference) error {
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return fmt.Errorf("%w: sqlite: failed to marshal value for key '%s': %v", userprefs.ErrSerialization, pref.Key, err)
	}

	defaultValueJSON, err := json.Marshal(pref.DefaultValue)
	if err != nil {
		// Handle nil DefaultValue gracefully: if it's nil, marshal it as SQL NULL / JSON null string
		if pref.DefaultValue == nil {
			defaultValueJSON = []byte("null")
		} else {
			return fmt.Errorf("%w: sqlite: failed to marshal default_value for key '%s': %v", userprefs.ErrSerialization, pref.Key, err)
		}
	}

	_, err = s.db.ExecContext(ctx, sqliteInsertSQL,
		pref.UserID,
		pref.Key,
		string(valueJSON),        // value for INSERT
		string(defaultValueJSON), // default_value for INSERT
		pref.Type,
		pref.Category,
		pref.UpdatedAt,           // updated_at for INSERT
		string(valueJSON),        // value for UPDATE
		string(defaultValueJSON), // default_value for UPDATE
		pref.UpdatedAt,           // updated_at for UPDATE
	)

	if err != nil {
		return fmt.Errorf("sqlite: failed to execute insert/update for user '%s', key '%s': %w", pref.UserID, pref.Key, err)
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
func (s *SQLiteStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query preferences by category for user '%s', category '%s': %w", userID, category, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(ctx, rows) // scanPreferences now handles rows.Close()
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
func (s *SQLiteStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query all preferences for user '%s': %w", userID, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(ctx, rows) // scanPreferences now handles rows.Close()
}

// Delete removes a specific preference for a given user ID and key.
// The provided context.Context can be used for cancellation or timeouts.
//
// Returns nil on successful deletion.
// If the preference to be deleted is not found (i.e., no rows affected by the SQL DELETE),
// it returns userprefs.ErrNotFound.
// If there's an issue with the database operation, a wrapped error is returned.
func (s *SQLiteStorage) Delete(ctx context.Context, userID, key string) error {
	result, err := s.db.ExecContext(ctx, sqliteDeleteSQL, userID, key)
	if err != nil {
		return fmt.Errorf("sqlite: failed to execute delete for user '%s', key '%s': %w", userID, key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// This error means we don't know if the delete succeeded or not, which is a problem.
		return fmt.Errorf("sqlite: failed to get affected rows for delete user '%s', key '%s': %w", userID, key, err)
	}

	if rowsAffected == 0 {
		return userprefs.ErrNotFound // No rows were deleted, so the preference was not found.
	}

	return nil
}

// Close closes the underlying SQLite database connection.
// It is important to call Close when the SQLiteStorage is no longer needed
// to release database resources, especially for file-based databases.
// For in-memory databases, closing might be less critical but still good practice.
//
// Returns an error if closing the connection fails.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
// It handles closing the rows object, returning any error from rows.Close()
// if no other error occurred, or wrapping it if another error was primary.
func (s *SQLiteStorage) scanPreferences(ctx context.Context, rows *sql.Rows) (prefsMap map[string]*userprefs.Preference, err error) {
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil {
			if err == nil { // If no primary error, the closeErr becomes the primary error
				err = fmt.Errorf("sqlite: failed to close rows: %w", closeErr)
			} else {
				// If there was already an error, wrap the original error with closeErr info.
				err = fmt.Errorf("sqlite: primary error: %w; additionally failed to close rows: %v", err, closeErr)
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
		var valueJSON string
		var defaultValueJSON sql.NullString
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
			err = fmt.Errorf("sqlite: failed to scan preference row: %w", scanErr)
			return
		}

		if category.Valid {
			pref.Category = category.String
		} else {
			pref.Category = "" // Or handle as appropriate for your application logic
		}

		if unmarshalErr := json.Unmarshal([]byte(valueJSON), &pref.Value); unmarshalErr != nil {
			err = fmt.Errorf("%w: sqlite: failed to unmarshal value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, unmarshalErr)
			return
		}

		if defaultValueJSON.Valid && defaultValueJSON.String != "null" {
			if unmarshalErr := json.Unmarshal([]byte(defaultValueJSON.String), &pref.DefaultValue); unmarshalErr != nil {
				err = fmt.Errorf("%w: sqlite: failed to unmarshal default_value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, unmarshalErr)
				return
			}
		} else {
			pref.DefaultValue = nil
		}

		prefsMap[pref.Key] = &pref
	}

	// Check for errors encountered during iteration.
	if iterationErr := rows.Err(); iterationErr != nil {
		err = fmt.Errorf("sqlite: error iterating preference rows: %w", iterationErr)
		return
	}

	return // prefsMap will be returned, err is nil or set by defer
}
