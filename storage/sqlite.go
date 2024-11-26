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
		INSERT INTO user_preferences (user_id, key, value, type, category, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, key) 
		DO UPDATE SET value = ?, updated_at = ?
	`

	sqliteSelectSQL = `
		SELECT user_id, key, value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = ? AND key = ?
	`

	sqliteSelectByCategorySQL = `
		SELECT user_id, key, value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = ? AND category = ?
	`

	sqliteSelectAllSQL = `
		SELECT user_id, key, value, type, category, updated_at 
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
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs the necessary database migrations.
func (s *SQLiteStorage) migrate() error {
	_, err := s.db.Exec(sqliteCreateTableSQL)
	return err
}

// Get retrieves a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
func (s *SQLiteStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON string

	err := s.db.QueryRowContext(ctx, sqliteSelectSQL, userID, key).Scan(
		&pref.UserID,
		&pref.Key,
		&valueJSON,
		&pref.Type,
		&pref.Category,
		&pref.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, userprefs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}

	if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &pref, nil
}

// Set stores or updates a preference.
// It marshals the value to JSON before storing.
func (s *SQLiteStorage) Set(ctx context.Context, pref *userprefs.Preference) error {
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	_, err = s.db.ExecContext(ctx, sqliteInsertSQL,
		pref.UserID,
		pref.Key,
		string(valueJSON),
		pref.Type,
		pref.Category,
		pref.UpdatedAt,
		string(valueJSON),
		pref.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}

	return nil
}

// GetByCategory retrieves all preferences for a user within a specific category.
func (s *SQLiteStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			// Log the error or handle it as needed
			fmt.Printf("Error closing rows: %v\n", cerr)
		}
	}()

	return s.scanPreferences(rows)
}

// GetAll retrieves all preferences for a user.
func (s *SQLiteStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, sqliteSelectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			// Log the error or handle it as needed
			fmt.Printf("Error closing rows: %v\n", cerr)
		}
	}()

	return s.scanPreferences(rows)
}

// Delete removes a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
func (s *SQLiteStorage) Delete(ctx context.Context, userID, key string) error {
	result, err := s.db.ExecContext(ctx, sqliteDeleteSQL, userID, key)
	if err != nil {
		return fmt.Errorf("failed to delete preference: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rows == 0 {
		return userprefs.ErrNotFound
	}

	return nil
}

// Close closes the SQLite database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
func (s *SQLiteStorage) scanPreferences(rows *sql.Rows) (map[string]*userprefs.Preference, error) {
	prefs := make(map[string]*userprefs.Preference)

	for rows.Next() {
		var pref userprefs.Preference
		var valueJSON string

		err := rows.Scan(
			&pref.UserID,
			&pref.Key,
			&valueJSON,
			&pref.Type,
			&pref.Category,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}

		if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}

		prefs[pref.Key] = &pref
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return prefs, nil
}
