// storage/postgres.go
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/CreativeUnicorns/userprefs"
	_ "github.com/lib/pq"
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

type PostgresStorage struct {
	db *sql.DB
}

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

func (s *PostgresStorage) migrate() error {
	_, err := s.db.Exec(createTableSQL)
	return err
}

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
	)

	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetByCategory(ctx context.Context, userID, category string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectByCategorySQL, userID, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer rows.Close()

	return s.scanPreferences(rows)
}

func (s *PostgresStorage) GetAll(ctx context.Context, userID string) (map[string]*userprefs.Preference, error) {
	rows, err := s.db.QueryContext(ctx, selectAllSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query preferences: %w", err)
	}
	defer rows.Close()

	return s.scanPreferences(rows)
}

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

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

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
