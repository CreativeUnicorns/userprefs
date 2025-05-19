package storage

import (
	"context"
	"database/sql"
	"database/sql/driver" // Added for driver.ErrBadConn
	"regexp"

	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock" // For sql.ErrConnDone

	"github.com/CreativeUnicorns/userprefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unmarshallable is a type that cannot be marshaled to JSON.
type unmarshallable struct {
	C chan int
}

// setupSQLiteTest creates a new SQLite database for testing and returns the storage and a cleanup function.
func setupSQLiteTest(t *testing.T) (*SQLiteStorage, func()) {
	t.Helper()
	dbPath := fmt.Sprintf("test_prefs_%s_%d.db", t.Name(), time.Now().UnixNano())
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err, "Failed to initialize SQLiteStorage")

	cleanup := func() {
		require.NoError(t, storage.Close(), "Failed to close storage")
		require.NoError(t, os.Remove(dbPath), "Failed to remove test database")
	}
	return storage, cleanup
}

func TestSQLiteStorage_Get(t *testing.T) {
	storage, cleanup := setupSQLiteTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "user_get_tests"
	testTime := time.Now().Truncate(time.Millisecond)

	t.Run("successful_get_with_and_without_default_value", func(t *testing.T) {
		key1 := "get_key_with_default"
		pref1 := &userprefs.Preference{
			UserID:       userID,
			Key:          key1,
			Value:        "actual_value_1",
			DefaultValue: "default_value_1",
			Type:         "string",
			Category:     "cat1",
			UpdatedAt:    testTime,
		}
		err := storage.Set(ctx, pref1)
		require.NoError(t, err)

		retrieved1, err := storage.Get(ctx, userID, key1)
		require.NoError(t, err)
		assert.Equal(t, pref1.UserID, retrieved1.UserID)
		assert.Equal(t, pref1.Key, retrieved1.Key)
		assert.Equal(t, pref1.Value, retrieved1.Value)
		assert.Equal(t, pref1.DefaultValue, retrieved1.DefaultValue)
		assert.Equal(t, pref1.Type, retrieved1.Type)
		assert.Equal(t, pref1.Category, retrieved1.Category)
		assert.Equal(t, pref1.UpdatedAt.Unix(), retrieved1.UpdatedAt.Unix())

		key2 := "get_key_without_default"
		pref2 := &userprefs.Preference{
			UserID:       userID,
			Key:          key2,
			Value:        "actual_value_2",
			DefaultValue: nil,
			Type:         "string",
			Category:     "cat2",
			UpdatedAt:    testTime,
		}
		err = storage.Set(ctx, pref2)
		require.NoError(t, err)

		retrieved2, err := storage.Get(ctx, userID, key2)
		require.NoError(t, err)
		assert.Nil(t, retrieved2.DefaultValue)
		assert.Equal(t, pref2.Value, retrieved2.Value)

		key3 := "get_key_with_json_null_default"
		pref3 := &userprefs.Preference{
			UserID:       userID,
			Key:          key3,
			Value:        "actual_value_3",
			DefaultValue: nil, // This will be marshaled to "null" by sqlite.go's Set
			Type:         "string",
			Category:     "cat3",
			UpdatedAt:    testTime,
		}
		err = storage.Set(ctx, pref3)
		require.NoError(t, err)

		retrieved3, err := storage.Get(ctx, userID, key3)
		require.NoError(t, err)
		assert.Nil(t, retrieved3.DefaultValue, "Expected DefaultValue to be nil when DB stores 'null'")
	})

	t.Run("get_non_existent_preference", func(t *testing.T) {
		_, err := storage.Get(ctx, userID, "non_existent_key_for_get_test")
		assert.ErrorIs(t, err, userprefs.ErrNotFound)
	})

	t.Run("json_unmarshal_value_error_on_get", func(t *testing.T) {
		key := "unmarshal_value_error_key"
		malformedJSONValue := "not_json_at_all" // Invalid JSON string for unmarshalling into interface{}
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			userID, key, malformedJSONValue, `"valid_default"`, "string", "test", testTime)
		require.NoError(t, err, "Failed to insert malformed value row directly")

		_, err = storage.Get(ctx, userID, key)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("sqlite: failed to unmarshal value for user '%s', key '%s'", userID, key)), "Error message mismatch: %v", err)
	})

	t.Run("json_unmarshal_default_value_error_on_get", func(t *testing.T) {
		key := "unmarshal_default_value_error_key"
		malformedJSONDefaultValue := "not_proper_json" // Invalid JSON string
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			userID, key, `"valid_value"`, malformedJSONDefaultValue, "string", "test", testTime)
		require.NoError(t, err, "Failed to insert malformed default_value row directly")

		_, err = storage.Get(ctx, userID, key)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("sqlite: failed to unmarshal default_value for user '%s', key '%s'", userID, key)), "Error message mismatch: %v", err)
	})
}

func TestSQLiteStorage_Set(t *testing.T) {
	storage, cleanup := setupSQLiteTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "user_set_tests"
	testTime := time.Now().Truncate(time.Millisecond) // Truncate for consistent comparison

	t.Run("successful_set_and_update", func(t *testing.T) {
		key := "theme_set_success"
		pref := &userprefs.Preference{
			UserID:       userID,
			Key:          key,
			Value:        "dark",
			DefaultValue: "light",
			Type:         "enum",
			Category:     "appearance",
			UpdatedAt:    testTime,
		}

		err := storage.Set(ctx, pref)
		require.NoError(t, err, "Set failed for initial preference")

		retrieved, err := storage.Get(ctx, userID, key)
		require.NoError(t, err, "Get failed after initial set")
		assert.Equal(t, "dark", retrieved.Value)
		assert.Equal(t, "light", retrieved.DefaultValue)
		assert.Equal(t, testTime.UnixMilli(), retrieved.UpdatedAt.UnixMilli()) // Compare UnixMilli for location insensitivity

		// Update preference
		updatedValue := "system"
		updatedDefaultValue := "blue"
		updatedTime := testTime.Add(time.Second)
		pref.Value = updatedValue
		pref.DefaultValue = updatedDefaultValue
		pref.UpdatedAt = updatedTime

		err = storage.Set(ctx, pref)
		require.NoError(t, err, "Set failed for preference update")

		retrievedAfterUpdate, err := storage.Get(ctx, userID, key)
		require.NoError(t, err, "Get failed after update")
		assert.Equal(t, updatedValue, retrievedAfterUpdate.Value)
		assert.Equal(t, updatedDefaultValue, retrievedAfterUpdate.DefaultValue)
		assert.Equal(t, updatedTime.UnixMilli(), retrievedAfterUpdate.UpdatedAt.UnixMilli()) // Compare UnixMilli
	})

	t.Run("json_marshal_default_value_error", func(t *testing.T) {
		pref := &userprefs.Preference{
			UserID:       userID,
			Key:          "theme_marshal_error_dv",
			Value:        "valid",
			DefaultValue: unmarshallable{C: make(chan int)}, // This will cause a marshal error
			Type:         "string",
			Category:     "test",
			UpdatedAt:    testTime,
		}

		err := storage.Set(ctx, pref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal default_value")
	})

	// Test case for json.Marshal error on Value (already present in sqlite.go Set, but good to have explicit test)
	t.Run("json_marshal_value_error", func(t *testing.T) {
		pref := &userprefs.Preference{
			UserID:       userID,
			Key:          "theme_marshal_error_v",
			Value:        unmarshallable{C: make(chan int)}, // This will cause a marshal error
			DefaultValue: "valid_default",
			Type:         "string",
			Category:     "test",
			UpdatedAt:    testTime,
		}

		err := storage.Set(ctx, pref)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal value")
	})
}

func TestSQLiteStorage_GetAll(t *testing.T) {
	storage, cleanup := setupSQLiteTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "user_getall_tests"
	testTime := time.Now().Truncate(time.Millisecond)

	// Setup some initial preferences for testing GetAll
	pref1 := &userprefs.Preference{
		UserID:       userID,
		Key:          "key1_for_getall",
		Value:        "value1",
		DefaultValue: "default1",
		Type:         "string",
		Category:     "catA",
		UpdatedAt:    testTime,
	}
	pref2 := &userprefs.Preference{
		UserID:       userID,
		Key:          "key2_for_getall",
		Value:        map[string]string{"subKey": "subValue"},
		DefaultValue: nil, // Will be stored as JSON null
		Type:         "object",
		Category:     "catB",
		UpdatedAt:    testTime.Add(time.Second),
	}
	require.NoError(t, storage.Set(ctx, pref1))
	require.NoError(t, storage.Set(ctx, pref2))

	t.Run("successful_getall", func(t *testing.T) {
		retrievedPrefs, err := storage.GetAll(ctx, userID)
		require.NoError(t, err)
		require.Len(t, retrievedPrefs, 2, "Expected to retrieve 2 preferences")

		rp1, ok := retrievedPrefs[pref1.Key]
		require.True(t, ok, "Preference 1 not found in GetAll result")
		assert.Equal(t, pref1.Value, rp1.Value)
		assert.Equal(t, pref1.DefaultValue, rp1.DefaultValue)

		rp2, ok := retrievedPrefs[pref2.Key]
		require.True(t, ok, "Preference 2 not found in GetAll result")
		// When unmarshaling into interface{}, JSON objects become map[string]interface{}
		expectedPref2Value := map[string]interface{}{"subKey": "subValue"}
		assert.Equal(t, expectedPref2Value, rp2.Value)
		assert.Nil(t, rp2.DefaultValue, "Expected DefaultValue to be nil for pref2")
	})

	t.Run("getall_non_existent_user", func(t *testing.T) {
		retrievedPrefs, err := storage.GetAll(ctx, "non_existent_user_for_getall")
		require.NoError(t, err)
		assert.Empty(t, retrievedPrefs, "Expected an empty map for a non-existent user")
	})

	t.Run("json_unmarshal_value_error_on_getall", func(t *testing.T) {
		// Insert a preference with a malformed value directly into the DB
		malformedUserID := "user_getall_malformed_value"
		keyGood := "good_key_malformed_value_user"
		keyMalformed := "malformed_value_key"

		// Insert a good preference first for this user
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedUserID, keyGood, `"good_value"`, `"good_default"`, "string", "test", testTime)
		require.NoError(t, err)

		// Now insert the malformed one
		_, err = storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedUserID, keyMalformed, "not_json_at_all", `"valid_default"`, "string", "test", testTime)
		require.NoError(t, err, "Failed to insert malformed value row directly")

		_, err = storage.GetAll(ctx, malformedUserID)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), "sqlite: failed to unmarshal value for key '"+keyMalformed+"' during scan"), "Error message mismatch: %v", err)
	})

	t.Run("json_unmarshal_default_value_error_on_getall", func(t *testing.T) {
		// Insert a preference with a malformed default_value directly into the DB
		malformedUserID := "user_getall_malformed_default"
		keyGood := "good_key_malformed_default_user"
		keyMalformedDefault := "malformed_default_value_key"
		keyMalformedDefaultValue := "malformed_default_value_key"

		// Insert a good preference first for this user
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedUserID, keyGood, `"good_value"`, `"good_default"`, "string", "test", testTime)
		require.NoError(t, err)

		// Now insert the one with malformed default
		_, err = storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedUserID, keyMalformedDefault, `"valid_value"`, "not_proper_json_for_default", "string", "test", testTime)
		require.NoError(t, err, "Failed to insert malformed default_value row directly")

		_, err = storage.GetAll(ctx, malformedUserID)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), "sqlite: failed to unmarshal default_value for key '"+keyMalformedDefaultValue+"' during scan"), "Error message mismatch: %v", err)
	})
}

func TestSQLiteStorage_Delete(t *testing.T) {
	storage, cleanup := setupSQLiteTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "user_delete_tests"
	key := "key_for_delete"
	testTime := time.Now().Truncate(time.Millisecond)

	// Set a preference to be deleted
	pref := &userprefs.Preference{
		UserID:       userID,
		Key:          key,
		Value:        "delete_me_value",
		DefaultValue: "delete_me_default",
		Type:         "string",
		Category:     "to_delete_cat",
		UpdatedAt:    testTime,
	}
	require.NoError(t, storage.Set(ctx, pref), "Failed to set preference for deletion test")

	// Ensure it's there before delete
	_, err := storage.Get(ctx, userID, key)
	require.NoError(t, err, "Preference not found before delete")

	// Delete preference
	err = storage.Delete(ctx, userID, key)
	require.NoError(t, err, "Delete failed")

	// Ensure deletion
	_, err = storage.Get(ctx, userID, key)
	assert.ErrorIs(t, err, userprefs.ErrNotFound, "Expected ErrNotFound after deletion")

	// Attempt to delete non-existent preference
	err = storage.Delete(ctx, userID, "non_existent_key_for_delete")
	assert.ErrorIs(t, err, userprefs.ErrNotFound, "Expected ErrNotFound when deleting non-existent key")
}

func TestSQLiteStorage_GetByCategory(t *testing.T) {
	storage, cleanup := setupSQLiteTest(t)
	defer cleanup()

	ctx := context.Background()
	userID := "user_getbycat_tests"
	testTime := time.Now().Truncate(time.Millisecond)

	categoryA := "categoryA_for_gbc"
	categoryB := "categoryB_for_gbc"

	// Setup preferences
	prefA1 := &userprefs.Preference{
		UserID:       userID,
		Key:          "keyA1",
		Value:        "valueA1",
		DefaultValue: "defaultA1",
		Type:         "string",
		Category:     categoryA,
		UpdatedAt:    testTime,
	}
	prefA2 := &userprefs.Preference{
		UserID:       userID,
		Key:          "keyA2",
		Value:        true,
		DefaultValue: nil, // Stored as JSON null
		Type:         "boolean",
		Category:     categoryA,
		UpdatedAt:    testTime.Add(time.Millisecond * 100),
	}
	prefB1 := &userprefs.Preference{
		UserID:       userID,
		Key:          "keyB1",
		Value:        123,
		DefaultValue: 0,
		Type:         "number",
		Category:     categoryB,
		UpdatedAt:    testTime.Add(time.Millisecond * 200),
	}
	require.NoError(t, storage.Set(ctx, prefA1))
	require.NoError(t, storage.Set(ctx, prefA2))
	require.NoError(t, storage.Set(ctx, prefB1))

	t.Run("successful_getbycategory", func(t *testing.T) {
		catAPrefs, err := storage.GetByCategory(ctx, userID, categoryA)
		require.NoError(t, err)
		require.Len(t, catAPrefs, 2, "Expected 2 preferences in categoryA")

		rpA1, ok := catAPrefs[prefA1.Key]
		require.True(t, ok)
		assert.Equal(t, prefA1.Value, rpA1.Value)
		assert.Equal(t, prefA1.DefaultValue, rpA1.DefaultValue)

		rpA2, ok := catAPrefs[prefA2.Key]
		require.True(t, ok)
		assert.Equal(t, prefA2.Value, rpA2.Value)
		assert.Nil(t, rpA2.DefaultValue)

		catBPrefs, err := storage.GetByCategory(ctx, userID, categoryB)
		require.NoError(t, err)
		require.Len(t, catBPrefs, 1, "Expected 1 preference in categoryB")
		rpB1, ok := catBPrefs[prefB1.Key]
		require.True(t, ok)
		// When unmarshaling into interface{}, JSON numbers become float64
		assert.Equal(t, float64(prefB1.Value.(int)), rpB1.Value)
	})

	t.Run("getbycategory_non_existent_category", func(t *testing.T) {
		prefs, err := storage.GetByCategory(ctx, userID, "non_existent_category_for_gbc")
		require.NoError(t, err)
		assert.Empty(t, prefs)
	})

	t.Run("getbycategory_non_existent_user", func(t *testing.T) {
		prefs, err := storage.GetByCategory(ctx, "non_existent_user_for_gbc", categoryA)
		require.NoError(t, err)
		assert.Empty(t, prefs)
	})

	t.Run("json_unmarshal_value_error_on_getbycategory", func(t *testing.T) {
		malformedCatUserID := "user_gbc_malformed_val"
		malformedCat := "malformed_value_cat_gbc"
		keyGood := "good_key_gbc_mv"
		keyMalformed := "malformed_val_key_gbc"

		// Insert a good preference in the category
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedCatUserID, keyGood, `"good_value"`, `"good_default"`, "string", malformedCat, testTime)
		require.NoError(t, err)

		// Insert the malformed one in the same category
		_, err = storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedCatUserID, keyMalformed, "not_json_at_all_gbc", `"valid_default"`, "string", malformedCat, testTime)
		require.NoError(t, err)

		_, err = storage.GetByCategory(ctx, malformedCatUserID, malformedCat)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), "sqlite: failed to unmarshal value for key '"+keyMalformed+"' during scan"), "Error message mismatch: %v", err)
	})

	t.Run("json_unmarshal_default_value_error_on_getbycategory", func(t *testing.T) {
		malformedCatUserID := "user_gbc_malformed_def"
		malformedCat := "malformed_default_cat_gbc"
		keyGood := "good_key_gbc_md"
		keyMalformedDefault := "malformed_def_key_gbc"

		// Insert a good preference in the category
		_, err := storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedCatUserID, keyGood, `"good_value"`, `"good_default"`, "string", malformedCat, testTime)
		require.NoError(t, err)

		// Insert the one with malformed default in the same category
		_, err = storage.db.ExecContext(ctx,
			"INSERT INTO user_preferences (user_id, key, value, default_value, type, category, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			malformedCatUserID, keyMalformedDefault, `"valid_value"`, "not_proper_json_for_default_gbc", "string", malformedCat, testTime)
		require.NoError(t, err)

		_, err = storage.GetByCategory(ctx, malformedCatUserID, malformedCat)
		require.Error(t, err)
		assert.True(t, errors.Is(err, userprefs.ErrSerialization), "expected ErrSerialization, got %v", err)
		assert.True(t, strings.Contains(err.Error(), "sqlite: failed to unmarshal default_value for key '"+keyMalformedDefault+"' during scan"), "Error message mismatch: %v", err)
	})
}

func TestSQLiteStorage_Concurrency(t *testing.T) {
	dbPath := "test_concurrency.db"
	defer func() {
		err := os.Remove(dbPath)
		if err != nil {
			t.Fatalf("Failed to remove test database: %v", err)
		}
	}()

	storage, err := NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize SQLiteStorage: %v", err)
	}
	defer func() {
		err := storage.Close()
		if err != nil {
			t.Fatalf("Close failed: %v", err)
		}
	}()

	ctx := context.Background()
	userID := "user_concurrent"
	key := "volume"

	// Define preference definition
	def := userprefs.PreferenceDefinition{
		Key:          "volume",
		Type:         "number",
		Category:     "audio",
		DefaultValue: 50,
	}

	// Set initial preference
	pref := &userprefs.Preference{
		UserID:    userID,
		Key:       key,
		Value:     50,
		Type:      def.Type,
		Category:  def.Category,
		UpdatedAt: time.Now(),
	}

	err = storage.Set(ctx, pref)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Concurrently update the preference
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(val int) {
			p := &userprefs.Preference{
				UserID:    userID,
				Key:       key,
				Value:     val,
				Type:      def.Type,
				Category:  def.Category,
				UpdatedAt: time.Now(),
			}
			if err := storage.Set(ctx, p); err != nil {
				t.Errorf("Set failed: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 100; i++ {
		<-done
	}

	// Get the final value
	finalPref, err := storage.Get(ctx, userID, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Assert that the final value is a float64, as SQLite returns numbers this way.
	// The actual value is non-deterministic due to concurrent writes,
	// so we only check the type and that no errors occurred during Set operations (checked in goroutines).
	_, ok := finalPref.Value.(float64)
	if !ok {
		t.Fatalf("Expected float64 value, got %T", finalPref.Value)
	}
}

// TestNewSQLiteStorage_ErrorPaths tests error scenarios for NewSQLiteStorage.
func TestNewSQLiteStorage_ErrorPaths(t *testing.T) {
	t.Run("sqlite_open_failure", func(t *testing.T) {
		originalSqliteOpen := sqliteOpen // Access the package-level var from sqlite.go
		expectedOpenErr := errors.New("mock sql.Open error")
		sqliteOpen = func(_, _ string) (*sql.DB, error) {
			return nil, expectedOpenErr
		}
		defer func() { sqliteOpen = originalSqliteOpen }() // Restore original

		_, err := NewSQLiteStorage("test_open_fail.db")
		require.Error(t, err, "NewSQLiteStorage should return an error when sqliteOpen fails")
		assert.True(t, errors.Is(err, expectedOpenErr), "Error should wrap the original sqliteOpen error. Got: %v", err)
		assert.Contains(t, err.Error(), "sqlite: failed to open database", "Error message mismatch")
	})

	t.Run("invalid_dsn_or_path_still_relevant_for_driver", func(t *testing.T) {
		// This test remains relevant as some DSN issues might be caught by the driver itself
		// even if our sqliteOpen wrapper is called.
		_, err := NewSQLiteStorage("/invalid\x00path/test.db") // \x00 is often problematic
		assert.Error(t, err, "Expected an error for an invalid DSN or path")
		assert.Contains(t, err.Error(), "sqlite:", "Error message should indicate a SQLite issue")

		// Test with an empty file path (not in-memory)
		_, err = NewSQLiteStorage("")
		assert.Error(t, err, "Expected an error for empty file path for non-in-memory DB")
		// This specific case might be caught before sqliteOpen if filePath is directly checked,
		// or by sqliteOpen if it passes the empty DSN to the actual sql.Open.
		// Based on NewSQLiteStorage logic, an empty filePath leads to "sqlite: file path cannot be empty for non-in-memory database"
		// which is before the sqliteOpen call.
		assert.Contains(t, err.Error(), "sqlite: filePath cannot be empty", "Error message mismatch for empty file path")
	})

	t.Run("ping_failure", func(t *testing.T) {
		originalSqliteOpen := sqliteOpen
		expectedPingErr := errors.New("mock PingContext error")

		sqliteOpen = func(_, _ string) (*sql.DB, error) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err, "Failed to create sqlmock")
			defer func() { _ = mock.ExpectationsWereMet() }() // Add deferred call
			mock.ExpectPing().WillReturnError(expectedPingErr)
			return db, nil
		}
		defer func() { sqliteOpen = originalSqliteOpen }()

		_, err := NewSQLiteStorage("test_ping_fail.db")
		require.Error(t, err, "NewSQLiteStorage should return an error when PingContext fails")
		assert.ErrorIs(t, err, expectedPingErr, "Error should wrap the original PingContext error")
		assert.Contains(t, err.Error(), "sqlite: failed to ping database", "Error message mismatch")
	})

	t.Run("migrate_failure", func(t *testing.T) {
		originalSqliteOpen := sqliteOpen
		expectedMigrateErr := errors.New("mock ExecContext error for migrate")

		sqliteOpen = func(_, _ string) (*sql.DB, error) {
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			require.NoError(t, err, "Failed to create sqlmock")
			defer func() { _ = mock.ExpectationsWereMet() }() // Add deferred call
			mock.ExpectPing()                                 // Expect successful ping
			mock.ExpectExec(regexp.QuoteMeta(sqliteCreateTableSQL)).WillReturnError(expectedMigrateErr)
			return db, nil
		}
		defer func() { sqliteOpen = originalSqliteOpen }()

		_, err := NewSQLiteStorage("test_migrate_fail.db")
		require.Error(t, err, "NewSQLiteStorage should return an error when migrate (ExecContext) fails")
		assert.ErrorIs(t, err, expectedMigrateErr, "Error should wrap the original migrate ExecContext error")
		assert.Contains(t, err.Error(), "sqlite: failed to run migrations", "Error message mismatch")
	})
}

func TestSQLiteStorage_CloseBehavior(t *testing.T) {
	// Setup a valid storage instance for these tests
	dbPath := fmt.Sprintf("test_close_behavior_%s_%d.db", t.Name(), time.Now().UnixNano())
	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err, "Failed to initialize SQLiteStorage for close tests")
	// No defer storage.Close() here as we are testing it.
	// No defer os.Remove(dbPath) here initially, will do it conditionally or at the end.

	t.Run("close_idempotent", func(t *testing.T) {
		err := storage.Close()
		assert.NoError(t, err, "First call to Close() should not return an error")
		err = storage.Close()
		assert.NoError(t, err, "Second call to Close() should also not return an error (idempotency)")
	})

	t.Run("operations_after_close", func(t *testing.T) {
		// Ensure storage is closed if not already by the previous subtest (though it should be)
		_ = storage.Close() // Call it again to be certain, should be harmless.

		ctx := context.Background()
		_, errGet := storage.Get(ctx, "user1", "key1")
		assert.Error(t, errGet, "Get operation after Close() should return an error")
		if !errors.Is(errGet, sql.ErrConnDone) && !errors.Is(errGet, driver.ErrBadConn) && !strings.Contains(errGet.Error(), "database is closed") {
			t.Errorf("Expected Get error after close to be sql.ErrConnDone, driver.ErrBadConn, or contain 'database is closed', got: %v", errGet)
		}

		pref := &userprefs.Preference{UserID: "user1", Key: "key1", Value: "value1"}
		errSet := storage.Set(ctx, pref)
		assert.Error(t, errSet, "Set operation after Close() should return an error")
		if !errors.Is(errSet, sql.ErrConnDone) && !errors.Is(errSet, driver.ErrBadConn) && !strings.Contains(errSet.Error(), "database is closed") {
			t.Errorf("Expected Set error after close to be sql.ErrConnDone, driver.ErrBadConn, or contain 'database is closed', got: %v", errSet)
		}

		errDel := storage.Delete(ctx, "user1", "key1")
		assert.Error(t, errDel, "Delete operation after Close() should return an error")
		if !errors.Is(errDel, sql.ErrConnDone) && !errors.Is(errDel, driver.ErrBadConn) && !strings.Contains(errDel.Error(), "database is closed") {
			t.Errorf("Expected Delete error after close to be sql.ErrConnDone, driver.ErrBadConn, or contain 'database is closed', got: %v", errDel)
		}

		_, errGetAll := storage.GetAll(ctx, "user1")
		assert.Error(t, errGetAll, "GetAll operation after Close() should return an error")
		if !errors.Is(errGetAll, sql.ErrConnDone) && !errors.Is(errGetAll, driver.ErrBadConn) && !strings.Contains(errGetAll.Error(), "database is closed") {
			t.Errorf("Expected GetAll error after close to be sql.ErrConnDone, driver.ErrBadConn, or contain 'database is closed', got: %v", errGetAll)
		}

		_, errGetByCat := storage.GetByCategory(ctx, "user1", "cat1")
		assert.Error(t, errGetByCat, "GetByCategory operation after Close() should return an error")
		if !errors.Is(errGetByCat, sql.ErrConnDone) && !errors.Is(errGetByCat, driver.ErrBadConn) && !strings.Contains(errGetByCat.Error(), "database is closed") {
			t.Errorf("Expected GetByCategory error after close to be sql.ErrConnDone, driver.ErrBadConn, or contain 'database is closed', got: %v", errGetByCat)
		}
	})

	// Cleanup the database file manually here since setupSQLiteTest's defer is not used.
	// We need to make sure it's closed before attempting to remove.
	_ = storage.Close() // Ensure it's closed
	errRemove := os.Remove(dbPath)
	if errRemove != nil && !os.IsNotExist(errRemove) { // Don't fail if already removed or never created properly
		t.Logf("Warning: Failed to remove test database %s: %v", dbPath, errRemove)
	}
}
