// Package storage provides a PostgreSQL-based implementation of the Storage interface.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/CreativeUnicorns/userprefs"
)

// sqlOpenFunc is a package-level variable that can be overridden for testing.
var sqlOpenFunc = sql.Open

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

// NewPostgresStorage initializes a new PostgresStorage instance.
// It connects to the PostgreSQL database using the provided connection string and runs migrations.
func NewPostgresStorage(connString string) (*PostgresStorage, error) {
	db, err := sqlOpenFunc("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close() // Attempt to close if ping fails
		return nil, fmt.Errorf("postgres: failed to ping database: %w", err)
	}

	storage := &PostgresStorage{db: db}
	if err := storage.migrate(); err != nil {
		db.Close() // Attempt to close if migration fails
		return nil, fmt.Errorf("postgres: failed to run migrations: %w", err)
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

// Get retrieves a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
func (s *PostgresStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON []byte
	var defaultValueJSON []byte // Added for DefaultValue

	err := s.db.QueryRowContext(ctx, selectSQL, userID, key).Scan(
		&pref.UserID,
		&pref.Key,
		&valueJSON,
		&defaultValueJSON, // Added for DefaultValue
		&pref.Type,
		&pref.Category,
		&pref.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, userprefs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to scan preference for user '%s', key '%s': %w", userID, key, err)
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

// Set stores or updates a preference.
// It marshals the value to JSON before storing.
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

// GetByCategory retrieves all preferences for a user within a specific category.
func (s *PostgresStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to query preferences by category for user '%s', category '%s': %w", userID, category, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(rows)
}

// GetAll retrieves all preferences for a user.
func (s *PostgresStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: failed to query all preferences for user '%s': %w", userID, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(rows)
}

// Delete removes a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
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

// Close closes the PostgreSQL database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
// It is the caller's responsibility to call rows.Close() if this function returns an error early.
// If this function returns nil error, it will close the rows.
func (s *PostgresStorage) scanPreferences(rows *sql.Rows) (map[string]*userprefs.Preference, error) {
	defer func() {
		if err := rows.Close(); err != nil {
			// Log this secondary error, e.g., using a logger if available
			// fmt.Printf("postgres: error closing rows in scanPreferences: %v\n", err)
		}
	}()

	prefs := make(map[string]*userprefs.Preference)

	for rows.Next() {
		var pref userprefs.Preference
		var valueJSON []byte
		var defaultValueJSON []byte // Stored as JSONB, can be null

		err := rows.Scan(
			&pref.UserID,
			&pref.Key,
			&valueJSON,
			&defaultValueJSON,
			&pref.Type,
			&pref.Category,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("postgres: failed to scan preference row: %w", err)
		}

		if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
			return nil, fmt.Errorf("%w: postgres: failed to unmarshal value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, err)
		}

		// Unmarshal DefaultValue, allow it to be null (represented by nil []byte or JSON 'null')
		if defaultValueJSON != nil && string(defaultValueJSON) != "null" {
			if err := json.Unmarshal(defaultValueJSON, &pref.DefaultValue); err != nil {
				return nil, fmt.Errorf("%w: postgres: failed to unmarshal default_value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, err)
			}
		} else {
			pref.DefaultValue = nil
		}

		prefs[pref.Key] = &pref
	}

	// Check for errors encountered during iteration.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: error iterating preference rows: %w", err)
	}

	return prefs, nil
}
