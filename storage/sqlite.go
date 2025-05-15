// Package storage provides a SQLite-based implementation of the Storage interface.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // SQLite driver

	"github.com/CreativeUnicorns/userprefs"
)

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

// SQLiteStorage implements the Storage interface using SQLite.
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage initializes a new SQLiteStorage instance.
// It connects to the SQLite database at the specified path and runs migrations.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to open database at %s: %w", dbPath, err)
	}

	if err := db.Ping(); err != nil {
		db.Close() // Attempt to close if ping fails
		return nil, fmt.Errorf("sqlite: failed to ping database at %s: %w", dbPath, err)
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.migrate(); err != nil {
		db.Close() // Attempt to close if migration fails
		return nil, fmt.Errorf("sqlite: failed to run migrations for %s: %w", dbPath, err)
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

// Get retrieves a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
func (s *SQLiteStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON string
	var defaultValueJSON sql.NullString // Added for DefaultValue

	err := s.db.QueryRowContext(ctx, sqliteSelectSQL, userID, key).Scan(
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
		return nil, fmt.Errorf("sqlite: failed to scan preference for user '%s', key '%s': %w", userID, key, err)
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

// Set stores or updates a preference.
// It marshals the value to JSON before storing.
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
		string(valueJSON),         // value for INSERT
		string(defaultValueJSON),  // default_value for INSERT
		pref.Type,
		pref.Category,
		pref.UpdatedAt,            // updated_at for INSERT
		string(valueJSON),         // value for UPDATE
		string(defaultValueJSON),  // default_value for UPDATE
		pref.UpdatedAt,            // updated_at for UPDATE
	)

	if err != nil {
		return fmt.Errorf("sqlite: failed to execute insert/update for user '%s', key '%s': %w", pref.UserID, pref.Key, err)
	}

	return nil
}

// GetByCategory retrieves all preferences for a user within a specific category.
func (s *SQLiteStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query preferences by category for user '%s', category '%s': %w", userID, category, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(rows) // scanPreferences now handles rows.Close()
}

// GetAll retrieves all preferences for a user.
func (s *SQLiteStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("sqlite: failed to query all preferences for user '%s': %w", userID, err)
	}
	// rows.Close() is deferred in scanPreferences, or will be called if scanPreferences returns an error early.
	return s.scanPreferences(rows) // scanPreferences now handles rows.Close()
}

// Delete removes a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
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

// Close closes the SQLite database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
// It is the caller's responsibility to call rows.Close() if this function returns an error early.
// If this function returns nil error, it will close the rows.
func (s *SQLiteStorage) scanPreferences(rows *sql.Rows) (map[string]*userprefs.Preference, error) {
	// Ensure rows are closed. If an error occurs during iteration, rows.Close() is called.
	// If iteration completes successfully, rows.Close() is also called.
	defer func() {
		if err := rows.Close(); err != nil {
			// This is a secondary error. The primary error (if any) from scanning/unmarshalling
			// has already been returned. For a library, one might log this. For now, we'll ignore it
			// if a primary error has already occurred, or panic if this is the only error and unexpected.
			// A simple Printf might be okay for dev, but not prod.
			// fmt.Printf("sqlite: error closing rows in scanPreferences: %v\n", err)
		}
	}()

	prefs := make(map[string]*userprefs.Preference)

	for rows.Next() {
		var pref userprefs.Preference
		var valueJSON string
		var defaultValueJSON sql.NullString

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
			return nil, fmt.Errorf("sqlite: failed to scan preference row: %w", err)
		}

		if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
			return nil, fmt.Errorf("%w: sqlite: failed to unmarshal value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, err)
		}

		if defaultValueJSON.Valid && defaultValueJSON.String != "null" {
			if err := json.Unmarshal([]byte(defaultValueJSON.String), &pref.DefaultValue); err != nil {
				return nil, fmt.Errorf("%w: sqlite: failed to unmarshal default_value for key '%s' during scan: %v", userprefs.ErrSerialization, pref.Key, err)
			}
		} else {
			pref.DefaultValue = nil
		}

		prefs[pref.Key] = &pref
	}

	// Check for errors encountered during iteration.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: error iterating preference rows: %w", err)
	}

	return prefs, nil
}
