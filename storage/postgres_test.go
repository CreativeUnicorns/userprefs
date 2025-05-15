package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CreativeUnicorns/userprefs"
)

// SQL query constants copied from postgres.go for precise matching
const (
	testCreateTableSQL = `
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

	testInsertSQL = `
		INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, key) 
		DO UPDATE SET value = $3, default_value = $4, updated_at = $7
	`

	testSelectSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND key = $2
	`

	testSelectByCategorySQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1 AND category = $2
	`

	testSelectAllSQL = `
		SELECT user_id, key, value, default_value, type, category, updated_at 
		FROM user_preferences 
		WHERE user_id = $1
	`

	testDeleteSQL = `
		DELETE FROM user_preferences 
		WHERE user_id = $1 AND key = $2
	`
)

// TestNewPostgresStorage tests the NewPostgresStorage constructor.
func TestNewPostgresStorage(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing()
		mock.ExpectExec(regexp.QuoteMeta(testCreateTableSQL)).WillReturnResult(sqlmock.NewResult(0, 0))

		originalSqlOpen := sqlOpenFunc // Use the package-level var from postgres.go
		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil // Return our mock DB
		}
		defer func() { sqlOpenFunc = originalSqlOpen }() // Restore original

		storage, err := NewPostgresStorage("dummy_conn_string")
		assert.NoError(t, err)
		assert.NotNil(t, storage)
		assert.NoError(t, mock.ExpectationsWereMet(), "sqlmock expectations not met")
	})

	t.Run("sql open error", func(t *testing.T) {
		expectedErr := errors.New("failed to open database")
		originalSqlOpen := sqlOpenFunc
		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			return nil, expectedErr
		}
		defer func() { sqlOpenFunc = originalSqlOpen }()

		_, err := NewPostgresStorage("dummy_conn_string")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, expectedErr), "Expected sql open error")
	})

	t.Run("ping error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("ping failed"))

		originalSqlOpen := sqlOpenFunc
		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { sqlOpenFunc = originalSqlOpen }()

		_, err = NewPostgresStorage("dummy_conn_string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to ping database")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("migrate error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing()
		mock.ExpectExec(regexp.QuoteMeta(testCreateTableSQL)).WillReturnError(errors.New("migrate failed"))

		originalSqlOpen := sqlOpenFunc
		sqlOpenFunc = func(driverName, dataSourceName string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { sqlOpenFunc = originalSqlOpen }()

		_, err = NewPostgresStorage("dummy_conn_string")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to run migrations")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func newTestPostgresStorage(t *testing.T) (*PostgresStorage, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	return &PostgresStorage{db: db}, mock // Pass the raw *sql.DB
}

func TestPostgresStorage_Set(t *testing.T) {
	storage, mock := newTestPostgresStorage(t)
	defer storage.Close()

	ctx := context.Background()
	testTime := time.Now().Truncate(time.Second)
	pref := &userprefs.Preference{
		UserID:       "user1",
		Key:          "theme",
		Value:        "dark",
		DefaultValue: "light", // Added DefaultValue
		Type:         "string",
		Category:     "appearance",
		UpdatedAt:    testTime,
	}
	valueJSON, err := json.Marshal(pref.Value)
	require.NoError(t, err)
	defaultValueJSON, err := json.Marshal(pref.DefaultValue) // Added for DefaultValue
	require.NoError(t, err)

	t.Run("successful set", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testInsertSQL)).
			WithArgs(pref.UserID, pref.Key, valueJSON, defaultValueJSON, pref.Type, pref.Category, pref.UpdatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := storage.Set(ctx, pref)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("json marshal value error", func(t *testing.T) {
		invalidPref := &userprefs.Preference{
			UserID: "user1",
			Key:    "baddata_value",
			Value:  make(chan int), // Cannot be marshalled
			DefaultValue: "some default",
		}
		err := storage.Set(ctx, invalidPref)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal value")
	})

	t.Run("json marshal default_value error", func(t *testing.T) {
		invalidPref := &userprefs.Preference{
			UserID: "user1",
			Key:    "baddata_default_value",
			Value:  "good value",
			DefaultValue: make(chan int), // Cannot be marshalled
		}
		err := storage.Set(ctx, invalidPref)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal default_value")
	})

	t.Run("db exec error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testInsertSQL)).
			WithArgs(pref.UserID, pref.Key, valueJSON, defaultValueJSON, pref.Type, pref.Category, pref.UpdatedAt).
			WillReturnError(errors.New("db exec error"))

		err := storage.Set(ctx, pref)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to execute insert/update for user 'user1', key 'theme'")
		assert.Contains(t, err.Error(), "db exec error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_Get(t *testing.T) {
	storage, mock := newTestPostgresStorage(t)
	defer storage.Close()

	ctx := context.Background()
	userID := "user1"
	key := "theme"
	testTime := time.Now().Truncate(time.Second)
	expectedValue := "dark"
	expectedDefaultValue := "light" // Added expected DefaultValue
	valueJSON, err := json.Marshal(expectedValue)
	require.NoError(t, err)
	defaultValueJSON, err := json.Marshal(expectedDefaultValue) // Added for DefaultValue
	require.NoError(t, err)

	t.Run("successful get", func(t *testing.T) {
		// Note: column order must match testSelectSQL
		rows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, key, valueJSON, defaultValueJSON, "string", "appearance", testTime)
		mock.ExpectQuery(regexp.QuoteMeta(testSelectSQL)).
			WithArgs(userID, key).
			WillReturnRows(rows)

		retPref, err := storage.Get(ctx, userID, key)
		assert.NoError(t, err)
		require.NotNil(t, retPref)
		assert.Equal(t, userID, retPref.UserID)
		assert.Equal(t, key, retPref.Key)
		assert.Equal(t, expectedValue, retPref.Value)
		assert.Equal(t, expectedDefaultValue, retPref.DefaultValue) // Assert DefaultValue
		assert.Equal(t, "string", retPref.Type)
		assert.Equal(t, "appearance", retPref.Category)
		assert.Equal(t, testTime, retPref.UpdatedAt.Truncate(time.Second))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get not found", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(testSelectSQL)).
			WithArgs(userID, "nonexistentkey").
			WillReturnError(sql.ErrNoRows)

		_, err := storage.Get(ctx, userID, "nonexistentkey")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Expected ErrNotFound, got %v", err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db query error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(testSelectSQL)).
			WithArgs(userID, key).
			WillReturnError(errors.New("db query error"))

		_, err := storage.Get(ctx, userID, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to scan preference for user 'user1', key 'theme'")
		assert.Contains(t, err.Error(), "db query error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("json unmarshal value error", func(t *testing.T) {
		malformedValueJSON := []byte("this is not json value")
		// default_value can be valid here as we are testing value unmarshal error
		rows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, key, malformedValueJSON, defaultValueJSON, "string", "appearance", testTime)
		mock.ExpectQuery(regexp.QuoteMeta(testSelectSQL)).
			WithArgs(userID, key).
			WillReturnRows(rows)

		_, err := storage.Get(ctx, userID, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal value")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("json unmarshal default_value error", func(t *testing.T) {
		malformedDefaultValueJSON := []byte("this is not json default_value")
		// value can be valid here
		rows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, key, valueJSON, malformedDefaultValueJSON, "string", "appearance", testTime)
		mock.ExpectQuery(regexp.QuoteMeta(testSelectSQL)).
			WithArgs(userID, key).
			WillReturnRows(rows)

		_, err := storage.Get(ctx, userID, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal default_value")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_Delete(t *testing.T) {
	storage, mock := newTestPostgresStorage(t)
	defer storage.Close()

	ctx := context.Background()
	userID := "user1"
	key := "theme"

	t.Run("successful delete", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testDeleteSQL)).
			WithArgs(userID, key).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := storage.Delete(ctx, userID, key)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("delete not found", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testDeleteSQL)).
			WithArgs(userID, "nonexistentkey").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := storage.Delete(ctx, userID, "nonexistentkey")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Expected ErrNotFound, got %v", err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("db exec error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testDeleteSQL)).
			WithArgs(userID, key).
			WillReturnError(errors.New("db delete error"))

		err := storage.Delete(ctx, userID, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to execute delete for user 'user1', key 'theme'")
		assert.Contains(t, err.Error(), "db delete error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected error", func(t *testing.T) {
		mock.ExpectExec(regexp.QuoteMeta(testDeleteSQL)).
			WithArgs(userID, key).
			WillReturnResult(sqlmock.NewErrorResult(errors.New("result error")))

		err := storage.Delete(ctx, userID, key)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get affected rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func createMockPrefRows(t *testing.T, userID string, prefs ...*userprefs.Preference) *sqlmock.Rows {
	rowNames := []string{"user_id", "key", "value", "type", "category", "updated_at"}
	rows := sqlmock.NewRows(rowNames)
	for _, p := range prefs {
		valueJSON, err := json.Marshal(p.Value)
		require.NoError(t, err)
		rows.AddRow(p.UserID, p.Key, valueJSON, p.Type, p.Category, p.UpdatedAt)
	}
	return rows
}

func TestPostgresStorage_GetAll(t *testing.T) {
	storage, mock := newTestPostgresStorage(t)
	defer storage.Close()

	ctx := context.Background()
	userID := "user1"
	testTime := time.Now().Truncate(time.Second)

	pref1 := &userprefs.Preference{UserID: userID, Key: "theme", Value: "dark", DefaultValue: "light", Type: "string", Category: "appearance", UpdatedAt: testTime}
	pref2 := &userprefs.Preference{UserID: userID, Key: "lang", Value: "en", DefaultValue: "fr", Type: "string", Category: "general", UpdatedAt: testTime}

	// Marshal pref values for mocking
	pref1ValueJSON, err := json.Marshal(pref1.Value)
	require.NoError(t, err)
	pref1DefaultValueJSON, err := json.Marshal(pref1.DefaultValue)
	require.NoError(t, err)
	pref2ValueJSON, err := json.Marshal(pref2.Value)
	require.NoError(t, err)
	pref2DefaultValueJSON, err := json.Marshal(pref2.DefaultValue)
	require.NoError(t, err)

	t.Run("successful getall", func(t *testing.T) {
		// Note: column order must match testSelectAllSQL
		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(pref1.UserID, pref1.Key, pref1ValueJSON, pref1DefaultValueJSON, pref1.Type, pref1.Category, pref1.UpdatedAt).
			AddRow(pref2.UserID, pref2.Key, pref2ValueJSON, pref2DefaultValueJSON, pref2.Type, pref2.Category, pref2.UpdatedAt)

		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnRows(mockRows)

		resultPrefs, err := storage.GetAll(ctx, userID)
		assert.NoError(t, err)
		require.NotNil(t, resultPrefs)
		assert.Len(t, resultPrefs, 2)
		// Check pref1
		assert.Contains(t, resultPrefs, pref1.Key)
		require.NotNil(t, resultPrefs[pref1.Key])
		assert.Equal(t, pref1.Value, resultPrefs[pref1.Key].Value)
		assert.Equal(t, pref1.DefaultValue, resultPrefs[pref1.Key].DefaultValue)
		assert.Equal(t, pref1.Type, resultPrefs[pref1.Key].Type)
		assert.Equal(t, pref1.Category, resultPrefs[pref1.Key].Category)
		// Check pref2
		assert.Contains(t, resultPrefs, pref2.Key)
		require.NotNil(t, resultPrefs[pref2.Key])
		assert.Equal(t, pref2.Value, resultPrefs[pref2.Key].Value)
		assert.Equal(t, pref2.DefaultValue, resultPrefs[pref2.Key].DefaultValue)
		assert.Equal(t, pref2.Type, resultPrefs[pref2.Key].Type)
		assert.Equal(t, pref2.Category, resultPrefs[pref2.Key].Category)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getall no preferences", func(t *testing.T) {
		emptyRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"})
		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnRows(emptyRows)

		resultPrefs, err := storage.GetAll(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, resultPrefs)
		assert.Len(t, resultPrefs, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getall db query error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnError(errors.New("db getall error"))

		_, err := storage.GetAll(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to query all preferences for user 'user1'")
		assert.Contains(t, err.Error(), "db getall error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("getall rows scan error", func(t *testing.T) {
		dummyDefaultValueJSON, err := json.Marshal("default")
		require.NoError(t, err)
		rowsWithError := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", []byte(`"value1"`), dummyDefaultValueJSON, "string", "cat1", testTime)
		rowsWithError.CloseError(errors.New("rows iteration error")) 

		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnRows(rowsWithError)

		_, err = storage.GetAll(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: error iterating preference rows") 
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getall json unmarshal value error in loop", func(t *testing.T) {
		malformedValueJSON := []byte("this is not json value")
		validValue1JSON, _ := json.Marshal("validValue1")
		defaultValue1JSON, _ := json.Marshal("default1")
		defaultValue2JSON, _ := json.Marshal("default2")

		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", validValue1JSON, defaultValue1JSON, "string", "cat1", testTime). 
			AddRow(userID, "key2", malformedValueJSON, defaultValue2JSON, "string", "cat2", testTime)

		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnRows(mockRows)

		_, err := storage.GetAll(ctx, userID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization")
		assert.Contains(t, err.Error(), "postgres: failed to unmarshal value for key 'key2' during scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getall json unmarshal default_value error in loop", func(t *testing.T) {
		malformedDefaultValueJSON := []byte("this is not json default")
		validValue1JSON, _ := json.Marshal("validValue1")
		validValue2JSON, _ := json.Marshal("validValue2")
		validDefaultValue1JSON, _ := json.Marshal("validDefault1")

		// For key2, value is valid, default_value is malformed.
		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", validValue1JSON, validDefaultValue1JSON, "string", "cat1", testTime). 
			AddRow(userID, "key2", validValue2JSON, malformedDefaultValueJSON, "string", "cat2", testTime)

		mock.ExpectQuery(regexp.QuoteMeta(testSelectAllSQL)).
			WithArgs(userID).
			WillReturnRows(mockRows)

		_, err := storage.GetAll(ctx, userID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization")
		assert.Contains(t, err.Error(), "postgres: failed to unmarshal default_value for key 'key2' during scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresStorage_GetByCategory(t *testing.T) {
	storage, mock := newTestPostgresStorage(t)
	defer storage.Close()

	ctx := context.Background()
	userID := "user1"
	category := "appearance"
	testTime := time.Now().Truncate(time.Second)

	pref1 := &userprefs.Preference{UserID: userID, Key: "theme", Value: "dark", DefaultValue: "light", Type: "string", Category: category, UpdatedAt: testTime}
	pref2 := &userprefs.Preference{UserID: userID, Key: "font", Value: "arial", DefaultValue: "sans-serif", Type: "string", Category: category, UpdatedAt: testTime}

	// Marshal pref values for mocking
	pref1ValueJSON, err := json.Marshal(pref1.Value)
	require.NoError(t, err)
	pref1DefaultValueJSON, err := json.Marshal(pref1.DefaultValue)
	require.NoError(t, err)
	pref2ValueJSON, err := json.Marshal(pref2.Value)
	require.NoError(t, err)
	pref2DefaultValueJSON, err := json.Marshal(pref2.DefaultValue)
	require.NoError(t, err)

	t.Run("successful getbycategory", func(t *testing.T) {
		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(pref1.UserID, pref1.Key, pref1ValueJSON, pref1DefaultValueJSON, pref1.Type, pref1.Category, pref1.UpdatedAt).
			AddRow(pref2.UserID, pref2.Key, pref2ValueJSON, pref2DefaultValueJSON, pref2.Type, pref2.Category, pref2.UpdatedAt)
		
		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, category).
			WillReturnRows(mockRows)

		resultPrefs, err := storage.GetByCategory(ctx, userID, category)
		assert.NoError(t, err)
		require.NotNil(t, resultPrefs)
		assert.Len(t, resultPrefs, 2)
		// Check pref1
		assert.Contains(t, resultPrefs, pref1.Key)
		require.NotNil(t, resultPrefs[pref1.Key])
		assert.Equal(t, pref1.Value, resultPrefs[pref1.Key].Value)
		assert.Equal(t, pref1.DefaultValue, resultPrefs[pref1.Key].DefaultValue)
		assert.Equal(t, pref1.Type, resultPrefs[pref1.Key].Type)
		assert.Equal(t, pref1.Category, resultPrefs[pref1.Key].Category)
		// Check pref2
		assert.Contains(t, resultPrefs, pref2.Key)
		require.NotNil(t, resultPrefs[pref2.Key])
		assert.Equal(t, pref2.Value, resultPrefs[pref2.Key].Value)
		assert.Equal(t, pref2.DefaultValue, resultPrefs[pref2.Key].DefaultValue)
		assert.Equal(t, pref2.Type, resultPrefs[pref2.Key].Type)
		assert.Equal(t, pref2.Category, resultPrefs[pref2.Key].Category)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getbycategory no preferences", func(t *testing.T) {
		emptyRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"})
		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, "nonexistent_category").
			WillReturnRows(emptyRows)

		resultPrefs, err := storage.GetByCategory(ctx, userID, "nonexistent_category")
		assert.NoError(t, err)
		assert.NotNil(t, resultPrefs)
		assert.Len(t, resultPrefs, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getbycategory db query error", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, category).
			WillReturnError(errors.New("db getbycategory error"))

		_, err := storage.GetByCategory(ctx, userID, category)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: failed to query preferences by category for user 'user1', category 'appearance'")
		assert.Contains(t, err.Error(), "db getbycategory error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	
	t.Run("getbycategory rows scan error", func(t *testing.T) {
		dummyDefaultValueJSON, err := json.Marshal("default")
		require.NoError(t, err)
		rowsWithError := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", []byte(`"value1"`), dummyDefaultValueJSON, "string", category, testTime)
		rowsWithError.CloseError(errors.New("rows iteration error for category"))

		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, category).
			WillReturnRows(rowsWithError)

		_, err = storage.GetByCategory(ctx, userID, category)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "postgres: error iterating preference rows")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getbycategory json unmarshal value error in loop", func(t *testing.T) {
		malformedValueJSON := []byte("this is not json value")
		validValueJSON, _ := json.Marshal("validValue")
		defaultValue1JSON, _ := json.Marshal("default1")
		defaultValue2JSON, _ := json.Marshal("default2")

		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", validValueJSON, defaultValue1JSON, "string", category, testTime). 
			AddRow(userID, "key2", malformedValueJSON, defaultValue2JSON, "string", category, testTime)    

		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, category).
			WillReturnRows(mockRows)

		_, err := storage.GetByCategory(ctx, userID, category)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization")
		assert.Contains(t, err.Error(), "postgres: failed to unmarshal value for key 'key2' during scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("getbycategory json unmarshal default_value error in loop", func(t *testing.T) {
		malformedDefaultValueJSON := []byte("this is not json default")
		validValueJSON, _ := json.Marshal("validValue")
		validDefaultValue1JSON, _ := json.Marshal("validDefault1")
		// For key2, value is valid, default_value is malformed.
		mockRows := sqlmock.NewRows([]string{"user_id", "key", "value", "default_value", "type", "category", "updated_at"}).
			AddRow(userID, "key1", validValueJSON, validDefaultValue1JSON, "string", category, testTime). 
			AddRow(userID, "key2", validValueJSON, malformedDefaultValueJSON, "string", category, testTime)

		mock.ExpectQuery(regexp.QuoteMeta(testSelectByCategorySQL)).
			WithArgs(userID, category).
			WillReturnRows(mockRows)

		_, err := storage.GetByCategory(ctx, userID, category)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization")
		assert.Contains(t, err.Error(), "postgres: failed to unmarshal default_value for key 'key2' during scan")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
