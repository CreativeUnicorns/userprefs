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
		INSERT INTO user_preferences (user_id, key, value, type, category, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, key) 
		DO UPDATE SET value = $3, updated_at = $6
	`

	selectSQL = `
		SELECT user_id, key, value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND key = $2
	`

	selectByCategorySQL = `
		SELECT user_id, key, value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND category = $2
	`

	selectAllSQL = `
		SELECT user_id, key, value, type, category, updated_at 
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
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	storage := &PostgresStorage{db: db}
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// migrate runs the necessary database migrations.
func (s *PostgresStorage) migrate() error {
	_, err := s.db.Exec(createTableSQL)
	return err
}

// Get retrieves a preference by user ID and key.
// It returns ErrNotFound if the preference does not exist.
func (s *PostgresStorage) Get(ctx context.Context, userID, key string) (*userprefs.Preference, error) {
	var pref userprefs.Preference
	var valueJSON []byte

	err := s.db.QueryRowContext(ctx, selectSQL, userID, key).Scan(
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

	if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &pref, nil
}

// Set stores or updates a preference.
// It marshals the value to JSON before storing.
func (s *PostgresStorage) Set(ctx context.Context, pref *userprefs.Preference) error {
	valueJSON, err := json.Marshal(pref.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	_, err = s.db.ExecContext(ctx, insertSQL,
		pref.UserID,
		pref.Key,
		valueJSON,
		pref.Type,
		pref.Category,
		pref.UpdatedAt,
		pref.Value,
		pref.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}

	return nil
}

// GetByCategory retrieves all preferences for a user within a specific category.
func (s *PostgresStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectByCategorySQL, userID, category)
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
func (s *PostgresStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSQL, userID)
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
func (s *PostgresStorage) Delete(ctx context.Context, userID, key string) error {
	result, err := s.db.ExecContext(ctx, deleteSQL, userID, key)
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

// Close closes the PostgreSQL database connection.
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// scanPreferences scans rows and constructs a map of preferences.
func (s *PostgresStorage) scanPreferences(rows *sql.Rows) (map[string]*userprefs.Preference, error) {
	prefs := make(map[string]*userprefs.Preference)

	for rows.Next() {
		var pref userprefs.Preference
		var valueJSON []byte

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

		if err := json.Unmarshal(valueJSON, &pref.Value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}

		prefs[pref.Key] = &pref
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return prefs, nil
}
