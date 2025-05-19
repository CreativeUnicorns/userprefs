package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	require.NotNil(t, storage, "NewMemoryStorage() should not return nil")
	// storage.items is unexported, so we cannot directly assert its initialization here.
	// The functionality of NewMemoryStorage is implicitly tested by subsequent Set/Get operations.
}

func TestMemoryStorage_Close(t *testing.T) {
	storage := NewMemoryStorage()
	require.NotNil(t, storage)

	t.Run("idempotent_close", func(t *testing.T) {
		err := storage.Close()
		assert.NoError(t, err, "First Close() should not return an error")
		err = storage.Close()
		assert.NoError(t, err, "Second Close() (idempotency) should not return an error")
	})

	// Since MemoryStorage.Close() currently does nothing more than return nil,
	// testing operations after close might not be meaningful unless its behavior changes.
	// If MemoryStorage were to, for example, nil out its map on Close or set a flag,
	// then we'd add tests here for operations after close.
	// For now, a simple Get after close might suffice to ensure it doesn't panic.
	t.Run("get_after_close", func(t *testing.T) {
		_ = storage.Close() // Ensure closed
		_, err := storage.Get(context.Background(), "user1", "key1")
		// Expect ErrNotFound because the item won't exist, and Close doesn't change this behavior.
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Get after Close() should return ErrNotFound for non-existent key")
	})
}

func TestMemoryStorage_Set_Get_Delete(t *testing.T) {
	storage := NewMemoryStorage()
	require.NotNil(t, storage)
	ctx := context.Background()

	testTime := time.Now().Truncate(time.Millisecond) // For consistent time comparison
	pref1 := &userprefs.Preference{
		UserID:       "user1",
		Key:          "key1",
		Value:        map[string]interface{}{"setting": "enabled"},
		Type:         "json", // Assuming 'json' for object types as per PreferenceDefinition doc
		Category:     "general",
		DefaultValue: map[string]interface{}{"setting": "disabled"},
		// CreatedAt is not part of the userprefs.Preference struct for Set operations;
		// storage manages timestamps internally.
		UpdatedAt: testTime, // This will be overwritten by Set operation
	}

	t.Run("set_and_get_preference", func(t *testing.T) {
		err := storage.Set(ctx, pref1)
		require.NoError(t, err, "Set should not return an error")

		retrievedPref, err := storage.Get(ctx, "user1", "key1")
		require.NoError(t, err, "Get should not return an error for existing key")
		require.NotNil(t, retrievedPref, "Retrieved preference should not be nil")

		assert.Equal(t, pref1.UserID, retrievedPref.UserID)
		assert.Equal(t, pref1.Key, retrievedPref.Key)
		assert.Equal(t, pref1.Value, retrievedPref.Value)
		assert.Equal(t, pref1.Type, retrievedPref.Type)
		assert.Equal(t, pref1.Category, retrievedPref.Category)
		assert.Equal(t, pref1.DefaultValue, retrievedPref.DefaultValue)
		// MemoryStorage.Set updates UpdatedAt. For a new item, it should be set.
		// For this test, we expect it to be at or after the Set call was made.
		// A more precise check would involve capturing time just before Set and checking retrievedPref.UpdatedAt.
		assert.False(t, retrievedPref.UpdatedAt.IsZero(), "UpdatedAt should be set")
		assert.True(t, retrievedPref.UpdatedAt.After(testTime) || retrievedPref.UpdatedAt.Equal(testTime), "UpdatedAt should be at or after the initial testTime used for pref1")
	})

	t.Run("get_non_existent_preference", func(t *testing.T) {
		_, err := storage.Get(ctx, "user1", "nonexistentkey")
		require.Error(t, err, "Get for non-existent key should return an error")
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Error should be ErrNotFound")
	})

	t.Run("update_existing_preference", func(t *testing.T) {
		updatedValue := map[string]interface{}{"setting": "disabled", "newField": 123}
		updatedPref := &userprefs.Preference{
			UserID:    "user1",
			Key:       "key1",
			Value:     updatedValue,
			Type:      "json", // Assuming 'json' for object types as per PreferenceDefinition doc
			Category:  "general_updated",
			UpdatedAt: time.Now().Truncate(time.Millisecond),
		}

		err := storage.Set(ctx, updatedPref)
		require.NoError(t, err, "Set (update) should not return an error")

		retrievedPref, err := storage.Get(ctx, "user1", "key1")
		require.NoError(t, err)
		assert.Equal(t, updatedValue, retrievedPref.Value)
		assert.Equal(t, "general_updated", retrievedPref.Category)
		// UpdatedAt should be later than the UpdatedAt of the initially set pref1,
		// which was captured *before* the first Set call in the previous subtest.
		// To be more robust, we'd get pref1 again before this update or record its UpdatedAt.
		// For now, check it's not zero and is after the initial testTime.
		assert.False(t, retrievedPref.UpdatedAt.IsZero(), "UpdatedAt on update should be set")
		assert.True(t, retrievedPref.UpdatedAt.After(testTime), "UpdatedAt after update should be after the initial testTime")
	})

	t.Run("delete_preference", func(t *testing.T) {
		err := storage.Delete(ctx, "user1", "key1")
		require.NoError(t, err, "Delete should not return an error for existing key")

		_, err = storage.Get(ctx, "user1", "key1")
		require.Error(t, err, "Get after delete should return an error")
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Error should be ErrNotFound after delete")
	})

	t.Run("delete_non_existent_preference_user_exists_key_not_exists", func(t *testing.T) {
		// Ensure user1 still has some prefs from previous tests or set one if necessary
		// For this test flow, user1 might be empty if "delete_preference" ran before it and deleted all of user1's prefs.
		// To be safe, let's ensure user1 exists with at least one pref if they were all deleted.
		_, getUserErr := storage.Get(ctx, "user1", "anyExistingKeyAfterDeletes") // Check if user1 has any keys left
		if errors.Is(getUserErr, userprefs.ErrNotFound) {
			// If user1 was cleared out by previous delete tests, set a dummy pref for them for this subtest to be meaningful
			tempPref := &userprefs.Preference{UserID: "user1", Key: "tempKey", Value: "tempVal", Type: "string"}
			setErr := storage.Set(ctx, tempPref)
			require.NoError(t, setErr, "Failed to set temporary preference for delete test")
		}

		err := storage.Delete(ctx, "user1", "nonexistentkey_todelete")
		require.Error(t, err, "Delete for non-existent key (user exists) should return an error")
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Error should be ErrNotFound when key does not exist for user")
	})

	t.Run("delete_non_existent_preference_user_not_exists", func(t *testing.T) {
		err := storage.Delete(ctx, "nonexistentuser", "anykey")
		require.Error(t, err, "Delete for non-existent user should return an error")
		assert.True(t, errors.Is(err, userprefs.ErrNotFound), "Error should be ErrNotFound when user does not exist")
	})
}

func TestMemoryStorage_GetAll_GetByCategory(t *testing.T) {
	storage := NewMemoryStorage()
	require.NotNil(t, storage)
	ctx := context.Background()

	prefUser1Cat1Key1 := &userprefs.Preference{UserID: "user1", Key: "key1", Category: "cat1", Value: "val1", Type: "string"}
	prefUser1Cat1Key2 := &userprefs.Preference{UserID: "user1", Key: "key2", Category: "cat1", Value: "val2", Type: "string"}
	prefUser1Cat2Key3 := &userprefs.Preference{UserID: "user1", Key: "key3", Category: "cat2", Value: "val3", Type: "string"}
	prefUser2Cat1Key4 := &userprefs.Preference{UserID: "user2", Key: "key4", Category: "cat1", Value: "val4", Type: "string"}

	prefsToSet := []*userprefs.Preference{
		prefUser1Cat1Key1,
		prefUser1Cat1Key2,
		prefUser1Cat2Key3,
		prefUser2Cat1Key4,
	}

	for _, p := range prefsToSet {
		err := storage.Set(ctx, p)
		require.NoError(t, err)
	}

	t.Run("get_all_for_user1", func(t *testing.T) {
		retrieved, err := storage.GetAll(ctx, "user1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Len(t, retrieved, 3, "User1 should have 3 preferences")
		assert.Equal(t, prefUser1Cat1Key1.Value, retrieved["key1"].Value)
		assert.Equal(t, prefUser1Cat1Key2.Value, retrieved["key2"].Value)
		assert.Equal(t, prefUser1Cat2Key3.Value, retrieved["key3"].Value)
	})

	t.Run("get_all_for_user2", func(t *testing.T) {
		retrieved, err := storage.GetAll(ctx, "user2")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Len(t, retrieved, 1, "User2 should have 1 preference")
		assert.Equal(t, prefUser2Cat1Key4.Value, retrieved["key4"].Value)
	})

	t.Run("get_all_for_non_existent_user", func(t *testing.T) {
		retrieved, err := storage.GetAll(ctx, "userNonExistent")
		require.NoError(t, err)
		assert.NotNil(t, retrieved, "GetAll for non-existent user should return a non-nil map")
		assert.Empty(t, retrieved, "GetAll for non-existent user should return an empty map")
	})

	t.Run("get_by_category_user1_cat1", func(t *testing.T) {
		retrieved, err := storage.GetByCategory(ctx, "user1", "cat1")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Len(t, retrieved, 2, "User1 cat1 should have 2 preferences")
		assert.Equal(t, prefUser1Cat1Key1.Value, retrieved["key1"].Value)
		assert.Equal(t, prefUser1Cat1Key2.Value, retrieved["key2"].Value)
	})

	t.Run("get_by_category_user1_cat2", func(t *testing.T) {
		retrieved, err := storage.GetByCategory(ctx, "user1", "cat2")
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Len(t, retrieved, 1, "User1 cat2 should have 1 preference")
		assert.Equal(t, prefUser1Cat2Key3.Value, retrieved["key3"].Value)
	})

	t.Run("get_by_category_user1_non_existent_cat", func(t *testing.T) {
		retrieved, err := storage.GetByCategory(ctx, "user1", "catNonExistent")
		require.NoError(t, err)
		assert.NotNil(t, retrieved, "GetByCategory for non-existent category should return a non-nil map")
		assert.Empty(t, retrieved, "GetByCategory for non-existent category should return an empty map")
	})

	t.Run("get_by_category_non_existent_user", func(t *testing.T) {
		retrieved, err := storage.GetByCategory(ctx, "userNonExistent", "cat1")
		require.NoError(t, err)
		assert.NotNil(t, retrieved, "GetByCategory for non-existent user should return a non-nil map")
		assert.Empty(t, retrieved, "GetByCategory for non-existent user should return an empty map")
	})
}
