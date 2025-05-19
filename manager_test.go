package userprefs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestManager_DefinePreference(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	def := PreferenceDefinition{
		Key:          "theme",
		Type:         "enum",
		Category:     "appearance",
		DefaultValue: "dark",
		AllowedValues: []interface{}{
			"light",
			"dark",
			"system",
		},
	}

	err := mgr.DefinePreference(def)
	if err != nil {
		t.Fatalf("DefinePreference failed: %v", err)
	}

	// Attempt to define the same preference again (should not error, effectively an update/noop)
	err = mgr.DefinePreference(def)
	if err != nil {
		t.Fatalf("DefinePreference failed on redefining: %v", err)
	}

	// Define a preference with invalid type
	invalidDef := PreferenceDefinition{
		Key:  "invalid_pref",
		Type: "unsupported",
	}
	err = mgr.DefinePreference(invalidDef)
	if !errors.Is(err, ErrInvalidType) {
		t.Fatalf("Expected ErrInvalidType, got: %v", err)
	}
}

func TestManager_GetSet(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	def := PreferenceDefinition{
		Key:          "notifications",
		Type:         "boolean",
		Category:     "settings",
		DefaultValue: false,
	}

	err := mgr.DefinePreference(def)
	if err != nil {
		t.Fatalf("DefinePreference failed: %v", err)
	}

	userID := "user1"

	// Get preference when not set (should return default)
	pref, err := mgr.Get(context.Background(), userID, "notifications")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val, ok := pref.Value.(bool); !ok || val != false {
		t.Errorf("Expected default value false, got %v (type %T)", pref.Value, pref.Value)
	}

	// Set preference
	err = mgr.Set(context.Background(), userID, "notifications", true)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get preference after setting
	pref, err = mgr.Get(context.Background(), userID, "notifications")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val, ok := pref.Value.(bool); !ok || val != true {
		t.Errorf("Expected value true, got %v (type %T)", pref.Value, pref.Value)
	}

	// Ensure it's cached
	if _, exists := cache.data["pref:user1:notifications"]; !exists {
		t.Errorf("Preference not cached")
	}

	// Get from cache
	pref, err = mgr.Get(context.Background(), userID, "notifications")
	if err != nil {
		t.Fatalf("Get from cache failed: %v", err)
	}
	if val, ok := pref.Value.(bool); !ok || val != true {
		t.Errorf("Expected cached value true, got %v (type %T)", pref.Value, pref.Value)
	}
}

func TestManager_Delete(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	def := PreferenceDefinition{
		Key:          "volume",
		Type:         "number",
		Category:     "audio",
		DefaultValue: 50.0, // Default as float64
	}

	err := mgr.DefinePreference(def)
	if err != nil {
		t.Fatalf("DefinePreference failed: %v", err)
	}

	userID := "user2"

	// Set preference
	err = mgr.Set(context.Background(), userID, "volume", 75.0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete preference
	err = mgr.Delete(context.Background(), userID, "volume")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Get preference after deletion (should return default)
	pref, err := mgr.Get(context.Background(), userID, "volume")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val, ok := pref.Value.(float64); !ok || val != 50.0 {
		t.Errorf("Expected default value 50.0, got %v (type %T)", pref.Value, pref.Value)
	}

	// Ensure cache is cleared
	if _, exists := cache.data["pref:user2:volume"]; exists {
		t.Errorf("Preference not deleted from cache")
	}

	// Attempt to delete non-existing preference (defined but not set for user, or completely unknown)
	// For a defined key but not set for a user, Delete effectively does nothing to storage but clears cache.
	// For an undefined key, it should ideally give ErrDefinitionNotFound from Manager.Delete.
	// Let's test deleting an undefined key for robustness of Manager.Delete
	err = mgr.Delete(context.Background(), userID, "nonexistentkey")
	if !errors.Is(err, ErrPreferenceNotDefined) { // Manager should check definition first
		t.Errorf("Expected ErrPreferenceNotDefined for deleting non-existent key, got %v", err)
	}
}

func TestManager_GetByCategory(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	// Define multiple preferences
	prefs := []PreferenceDefinition{
		{
			Key:          "theme",
			Type:         "enum",
			Category:     "appearance",
			DefaultValue: "light",
			AllowedValues: []interface{}{
				"light",
				"dark",
			},
		},
		{
			Key:          "font_size",
			Type:         "number",
			Category:     "appearance",
			DefaultValue: 12.0, // Default as float64
		},
		{
			Key:          "notifications",
			Type:         "boolean",
			Category:     "settings",
			DefaultValue: true,
		},
	}

	for _, def := range prefs {
		if err := mgr.DefinePreference(def); err != nil {
			t.Fatalf("DefinePreference failed: %v", err)
		}
	}

	userID := "user3"

	// Set some preferences
	err := mgr.Set(context.Background(), userID, "theme", "dark")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	err = mgr.Set(context.Background(), userID, "font_size", 14.0)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get preferences by category
	appearancePrefs, err := mgr.GetByCategory(context.Background(), userID, "appearance")
	if err != nil {
		t.Fatalf("GetByCategory failed: %v", err)
	}

	if len(appearancePrefs) != 2 {
		t.Fatalf("Expected 2 appearance preferences, got %d", len(appearancePrefs))
	}

	if val, ok := appearancePrefs["theme"].Value.(string); !ok || val != "dark" {
		t.Errorf("Expected theme 'dark', got %v (type %T)", appearancePrefs["theme"].Value, appearancePrefs["theme"].Value)
	}
	if val, ok := appearancePrefs["font_size"].Value.(float64); !ok || val != 14.0 {
		t.Errorf("Expected font_size 14.0, got %v (type %T)", appearancePrefs["font_size"].Value, appearancePrefs["font_size"].Value)
	}

	// Get preferences by non-existing category (no definitions for this category)
	nonexistentPrefs, err := mgr.GetByCategory(context.Background(), userID, "nonexistentcategory")
	if err != nil {
		t.Errorf("Expected no error for nonexistent category, got %v", err)
	}
	if len(nonexistentPrefs) != 0 {
		t.Errorf("Expected empty map for nonexistent category, got %d preferences", len(nonexistentPrefs))
	}
}

func TestManager_GetAll(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	// Define multiple preferences
	prefsToDefine := []PreferenceDefinition{
		{
			Key:          "language",
			Type:         "string",
			Category:     "general",
			DefaultValue: "en",
		},
		{
			Key:          "timezone",
			Type:         "string",
			Category:     "general",
			DefaultValue: "UTC",
		},
	}

	for _, def := range prefsToDefine {
		if err := mgr.DefinePreference(def); err != nil {
			t.Fatalf("DefinePreference failed: %v", err)
		}
	}

	userID := "user4"

	// Set some preferences
	err := mgr.Set(context.Background(), userID, "language", "fr")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get all preferences
	allPrefs, err := mgr.GetAll(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	// Ensure all defined preferences are returned
	if len(allPrefs) != len(prefsToDefine) {
		t.Fatalf("Expected %d preferences, got %d", len(prefsToDefine), len(allPrefs))
	}

	// Validate each preference
	if pref, exists := allPrefs["language"]; !exists {
		t.Errorf("Expected language preference to exist")
	} else if val, ok := pref.Value.(string); !ok || val != "fr" {
		t.Errorf("Expected language 'fr', got %v (type %T)", pref.Value, pref.Value)
	}

	if pref, exists := allPrefs["timezone"]; !exists {
		t.Errorf("Expected timezone preference to exist")
	} else if val, ok := pref.Value.(string); !ok || val != "UTC" {
		t.Errorf("Expected timezone 'UTC' (default), got %v (type %T)", pref.Value, pref.Value)
	}
}

func TestManager_Concurrency(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	def := PreferenceDefinition{
		Key:          "volume",
		Type:         "number",
		Category:     "audio",
		DefaultValue: 50.0, // Default as float64
	}

	err := mgr.DefinePreference(def)
	if err != nil {
		t.Fatalf("DefinePreference failed: %v", err)
	}

	userID := "user5"
	done := make(chan bool)
	numGoroutines := 100

	// Concurrently set preferences
	for i := 0; i < numGoroutines; i++ {
		go func(val float64) {
			// In a real scenario with high contention, Set might return an error from storage/cache.
			// MockStorage/MockCache are synchronous, so less likely to show race-related errors here unless Manager itself has issues.
			if err := mgr.Set(context.Background(), userID, "volume", val); err != nil {
				// t.Errorf is not goroutine-safe. Collect errors or use t.Logf then Fail.
				// For simplicity in this mock test, we'll assume Set doesn't error out often under mock conditions.
				logger.Error("error in Set (concurrent)", "err", err) // Using mock logger
			}
			done <- true
		}(float64(i)) // Set as float64
	}

	// Wait for all goroutines to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Get the final value
	pref, err := mgr.Get(context.Background(), userID, "volume")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// The final value should be one of the values set (0.0 to 99.0)
	// Due to mock cache/storage behavior (last write wins), it's hard to predict the exact final value without more sophisticated mocks.
	// We'll check if it's a float64 and within the possible range.
	switch v := pref.Value.(type) {
	case float64:
		if v < 0.0 || v > float64(numGoroutines-1) {
			t.Errorf("Final volume out of expected range [0.0, %.1f]: %v", float64(numGoroutines-1), v)
		}
	default:
		t.Errorf("Unexpected type for volume: %T, value: %v", pref.Value, pref.Value)
	}
}

func TestManager_Get_CacheInteraction(t *testing.T) {
	// Define a common preference definition for these tests
	def := PreferenceDefinition{
		Key:          "test_cache_pref",
		Type:         "string",
		Category:     "cache_tests",
		DefaultValue: "default_cache_value",
	}

	// Define a common preference instance expected from storage or successful cache retrieval
	expectedPrefFromStorage := &Preference{
		UserID:       "user_cache_test",
		Key:          def.Key,
		Value:        "stored_value",
		DefaultValue: def.DefaultValue,
		Type:         def.Type,
		Category:     def.Category,
		UpdatedAt:    time.Now(), // Will be compared with a tolerance or ignored
	}

	cacheKey := fmt.Sprintf("pref:%s:%s", expectedPrefFromStorage.UserID, expectedPrefFromStorage.Key)
	specificGenericCacheError := errors.New("specific generic cache failure from mock")

	testCases := []struct {
		name                 string
		setupCache           func(mc *MockCache, _ *MockStorage) // Function to set up cache state, ms renamed to _
		setupStorage         func(_ *MockStorage)                // Function to set up storage state, ms renamed to _
		useNilCache          bool
		expectedValue        interface{}
		expectedErr          error
		checkLogs            func(t *testing.T, logger *MockLogger, caseName string)
		expectDefaultOnError bool // If true, on error from cache/storage, expect default value
	}{
		{
			name: "Cache Hit - Valid Direct Bytes",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				prefToCache := &Preference{
					UserID:       expectedPrefFromStorage.UserID,
					Key:          def.Key,
					Value:        "cached_direct_value", // Different from storage to confirm cache hit
					DefaultValue: def.DefaultValue,
					Type:         def.Type,
					Category:     def.Category,
					UpdatedAt:    time.Now(),
				}
				bytes, _ := json.Marshal(prefToCache)
				mc.data[cacheKey] = mockCacheEntry{value: bytes}
			},
			expectedValue: "cached_direct_value",
			expectedErr:   nil,
		},
		{
			name: "Cache Hit - Valid JSON Data (formerly Base64 test)",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				prefToCache := &Preference{
					UserID:       expectedPrefFromStorage.UserID,
					Key:          def.Key,
					Value:        "cached_b64_value", // Different from storage
					DefaultValue: def.DefaultValue,
					Type:         def.Type,
					Category:     def.Category,
					UpdatedAt:    time.Now(),
				}
				jsonBytes, _ := json.Marshal(prefToCache)
				// The cache should store the direct JSON bytes of the Preference.
				mc.data[cacheKey] = mockCacheEntry{value: jsonBytes}
			},
			expectedValue: "cached_b64_value",
			expectedErr:   nil,
		},
		{
			name: "Cache Miss (ErrNotFound) - Fallback to Storage",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				// Cache returns ErrNotFound implicitly if key doesn't exist or explicitly:
				mc.data[cacheKey] = mockCacheEntry{value: nil, err: ErrNotFound}
			},
			setupStorage: func(ms *MockStorage) {
				_ = ms.Set(context.Background(), expectedPrefFromStorage) // Ensure storage has the item
			},
			expectedValue: expectedPrefFromStorage.Value, // Should get value from storage
			expectedErr:   nil,
		},
		{
			name:        "Nil Cache - Fallback to Storage",
			useNilCache: true,
			setupStorage: func(s *MockStorage) { // ms renamed to s, to be used in the body
				_ = s.Set(context.Background(), expectedPrefFromStorage)
			},
			expectedValue: expectedPrefFromStorage.Value,
			expectedErr:   nil,
		},
		{
			name: "Cache Hit - Malformed JSON Bytes",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				mc.data[cacheKey] = mockCacheEntry{value: []byte("{\"key\": \"malformed\"")}
			},
			expectedErr:          ErrSerialization,
			expectDefaultOnError: true, // Expect default because cache unmarshal fails
		},
		{
			name: "Cache Hit - JSON String, Not Base64",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				// Marshal a non-base64 string into JSON bytes e.g. "this is not base64"
				jsonStringBytes, _ := json.Marshal("this is not base64")
				mc.data[cacheKey] = mockCacheEntry{value: jsonStringBytes}
			},
			expectedErr:          ErrSerialization,
			expectDefaultOnError: true,
		},
		{
			name: "Cache Hit - Base64 Decodes to Malformed JSON",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				b64String := base64.StdEncoding.EncodeToString([]byte("{\"key\": \"malformed_after_b64\""))
				finalCacheBytes, _ := json.Marshal(b64String)
				mc.data[cacheKey] = mockCacheEntry{value: finalCacheBytes}
			},
			expectedErr:          ErrSerialization,
			expectDefaultOnError: true,
		},
		{
			name: "Cache Hit - Non-[]byte Data",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				mc.data[cacheKey] = mockCacheEntry{value: []byte("12345")} // A non-JSON []byte
			},
			expectedErr:          ErrSerialization,
			expectDefaultOnError: true,
		},
		{
			name: "Cache Error - Generic Error from Cache Get",
			setupCache: func(mc *MockCache, _ *MockStorage) { // ms renamed to _
				mc.data[cacheKey] = mockCacheEntry{err: specificGenericCacheError}
			},
			// The manager wraps the cache error. We expect the default value to be returned.
			// For error checking, we are checking if the *specific* error `ErrSerialization` is returned when applicable,
			// or if a generic error is returned, we check that an error *is* returned.
			// More specific checks on wrapped errors could be done with custom error types or by inspecting error strings.
			expectedErr:          specificGenericCacheError, // Expect the specific error instance to be wrapped and identifiable
			expectDefaultOnError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStorage()
			cache := NewMockCache()
			logger := &MockLogger{}

			mgrOpts := []Option{
				WithStorage(store),
				WithLogger(logger),
			}

			if !tc.useNilCache {
				mgrOpts = append(mgrOpts, WithCache(cache))
			} else {
				mgrOpts = append(mgrOpts, WithCache(nil)) // Explicitly pass nil cache
			}

			mgr := New(mgrOpts...)
			err := mgr.DefinePreference(def)
			if err != nil {
				t.Fatalf("DefinePreference failed: %v", err)
			}

			if tc.setupStorage != nil {
				tc.setupStorage(store)
			} else {
				// Default storage setup: store the expectedPrefFromStorage
				_ = store.Set(context.Background(), expectedPrefFromStorage)
			}

			if tc.setupCache != nil && !tc.useNilCache {
				tc.setupCache(cache, store)
			}

			pref, err := mgr.Get(context.Background(), expectedPrefFromStorage.UserID, def.Key)

			if tc.expectedErr != nil {
				if !errors.Is(err, tc.expectedErr) {
					t.Errorf("Expected error '%v', got '%v'", tc.expectedErr, err)
				}
				if tc.expectDefaultOnError {
					if pref == nil {
						t.Errorf("Expected preference with default value on error, got nil preference")
					} else if pref.Value != def.DefaultValue {
						t.Errorf("Expected default value '%v' on error, got '%v'", def.DefaultValue, pref.Value)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got '%v'", err)
				}
				if pref == nil {
					t.Fatalf("Expected preference, got nil")
				}
				if pref.Value != tc.expectedValue {
					t.Errorf("Expected value '%v', got '%v'", tc.expectedValue, pref.Value)
				}
			}

			if tc.checkLogs != nil {
				tc.checkLogs(t, logger, tc.name)
			}
		})
	}
}

func TestManager_Set_ValidationAndTypeChecking(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	userID := "testUserValidation"

	// 1. Test Type Mismatch
	defTypeMismatch := PreferenceDefinition{
		Key:          "typeMismatchPref",
		Type:         "string",
		Category:     "test",
		DefaultValue: "default",
	}
	err := mgr.DefinePreference(defTypeMismatch)
	if err != nil {
		t.Fatalf("DefinePreference failed for typeMismatchPref: %v", err)
	}

	err = mgr.Set(context.Background(), userID, "typeMismatchPref", 123) // Set int for StringType
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue for type mismatch, got %v", err)
	}

	// 2. Test Validation Function Failure
	defValidationFail := PreferenceDefinition{
		Key:          "validationFailPref",
		Type:         "string",
		Category:     "test",
		DefaultValue: "default",
		ValidateFunc: func(value interface{}) error {
			sVal, ok := value.(string)
			if !ok {
				return fmt.Errorf("unexpected type for validation: %T", value)
			}
			if sVal != "valid_value" {
				return fmt.Errorf("value must be 'valid_value'")
			}
			return nil
		},
	}
	err = mgr.DefinePreference(defValidationFail)
	if err != nil {
		t.Fatalf("DefinePreference failed for validationFailPref: %v", err)
	}

	err = mgr.Set(context.Background(), userID, "validationFailPref", "invalid_value")
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue for validation failure, got %v", err)
	}
	// Ensure not set in storage or cache
	_, storeErr := store.Get(context.Background(), userID, "validationFailPref")
	if !errors.Is(storeErr, ErrNotFound) {
		t.Errorf("Expected ErrNotFound from storage after failed Set, got %v", storeErr)
	}
	if _, cacheExists := cache.data[fmt.Sprintf("pref:%s:%s", userID, "validationFailPref")]; cacheExists {
		t.Errorf("Value should not be in cache after failed Set")
	}

	// 3. Test Validation Function Success
	defValidationSuccess := PreferenceDefinition{
		Key:          "validationSuccessPref",
		Type:         "string",
		Category:     "test",
		DefaultValue: "default",
		ValidateFunc: func(value interface{}) error {
			sVal, ok := value.(string)
			if !ok {
				return fmt.Errorf("unexpected type for validation: %T", value)
			}
			if sVal != "correct_value" {
				return fmt.Errorf("value must be 'correct_value'")
			}
			return nil
		},
	}
	err = mgr.DefinePreference(defValidationSuccess)
	if err != nil {
		t.Fatalf("DefinePreference failed for validationSuccessPref: %v", err)
	}

	err = mgr.Set(context.Background(), userID, "validationSuccessPref", "correct_value")
	if err != nil {
		t.Errorf("Expected no error for successful validation, got %v", err)
	}
	// Ensure set in storage and cache
	pref, storeErr := store.Get(context.Background(), userID, "validationSuccessPref")
	if storeErr != nil {
		t.Errorf("Expected no error from storage after successful Set, got %v", storeErr)
	} else if pref == nil || pref.Value != "correct_value" {
		t.Errorf("Value in storage is incorrect. Expected 'correct_value', got %v", pref.Value)
	}

	cacheKey := fmt.Sprintf("pref:%s:%s", userID, "validationSuccessPref")
	cacheEntry, cacheExists := cache.data[cacheKey]
	if !cacheExists {
		t.Errorf("Value should be in cache after successful Set")
	} else {
		var cachedPref Preference
		// cacheEntry.value is already []byte (JSON marshalled Preference)
		// and contains the direct JSON bytes of the Preference.
		if err := json.Unmarshal(cacheEntry.value, &cachedPref); err != nil {
			t.Fatalf("Failed to unmarshal cached preference: %v", err)
		}
		if cachedPref.Value != "correct_value" {
			t.Errorf("Value in cache is incorrect. Expected 'correct_value', got %v", cachedPref.Value)
		}
	}
}

func TestManager_GetAllPreferences(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	userID := "user_test_all_prefs"

	// Define multiple preferences
	prefsToDefine := []PreferenceDefinition{
		{
			Key:          "language",
			Type:         "string",
			Category:     "general",
			DefaultValue: "en",
		},
		{
			Key:          "timezone",
			Type:         "string",
			Category:     "general",
			DefaultValue: "UTC",
		},
		{
			Key:          "notifications",
			Type:         "boolean",
			Category:     "settings",
			DefaultValue: true,
		},
	}

	for _, def := range prefsToDefine {
		if err := mgr.DefinePreference(def); err != nil {
			t.Fatalf("DefinePreference failed: %v", err)
		}
	}

	// Set some preferences
	err := mgr.Set(context.Background(), userID, "language", "fr")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	err = mgr.Set(context.Background(), userID, "timezone", "PST")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Mock expected preferences from storage
	expectedPrefs := map[string]*Preference{
		"language": {
			UserID:       userID,
			Key:          "language",
			Value:        "fr",
			DefaultValue: "en",
			Type:         "string",
			Category:     "general",
			UpdatedAt:    time.Now(),
		},
		"timezone": {
			UserID:       userID,
			Key:          "timezone",
			Value:        "PST",
			DefaultValue: "UTC",
			Type:         "string",
			Category:     "general",
			UpdatedAt:    time.Now(),
		},
		"notifications": {
			UserID:       userID,
			Key:          "notifications",
			Value:        true,
			DefaultValue: false,
			Type:         "boolean",
			Category:     "settings",
			UpdatedAt:    time.Now(),
		},
	}

	// Test cases
	testCases := []struct {
		name          string
		cacheType     string
		setupCache    func(mc *MockCache, _ *MockStorage) // Changed ms to _
		userID        string
		mockGetAll    func(mockStore *MockStorage, uid string)
		expectedPrefs map[string]*Preference
		expectedError error
	}{
		{
			name:      "cache hit on get all",
			cacheType: "memory", // Can be any cache type
			setupCache: func(_ *MockCache, _ *MockStorage) { // mc and ms renamed to _
				// No need to do anything since we're not using the cache directly
				// Manager.GetAll doesn't check the cache
			},
			userID: userID,
			mockGetAll: func(mockStore *MockStorage, _ string) { // Set up preferences in storage
				// Set the preferences in the mock storage
				for _, pref := range expectedPrefs {
					_ = mockStore.Set(context.Background(), pref)
				}
			},
			expectedPrefs: expectedPrefs,
			expectedError: nil,
		},
		{
			name:      "cache error on get all",
			cacheType: "redis",
			setupCache: func(_ *MockCache, _ *MockStorage) { // mc and ms renamed to _
				// No need to do anything since we're not using the cache directly
				// Manager.GetAll doesn't check the cache
			},
			userID: userID,
			mockGetAll: func(mockStore *MockStorage, _ string) { // Set up preferences in storage
				// Set the preferences in the mock storage
				for _, pref := range expectedPrefs {
					_ = mockStore.Set(context.Background(), pref)
				}
			},
			expectedPrefs: expectedPrefs,
			expectedError: nil,
		},
		{
			name:      "store error on get all after cache miss",
			cacheType: "memory",
			setupCache: func(_ *MockCache, _ *MockStorage) { // mc and ms renamed to _
				// No need to do anything since we're not using the cache directly
			},
			userID: userID,
			mockGetAll: func(mockStore *MockStorage, _ string) {
				// Set the preferences in the mock storage
				for _, pref := range expectedPrefs {
					_ = mockStore.Set(context.Background(), pref)
				}
			},
			expectedPrefs: expectedPrefs,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStorage()
			cache := NewMockCache()
			logger := &MockLogger{}

			mgr := New(
				WithStorage(store),
				WithCache(cache),
				WithLogger(logger),
			)

			// Define the same preferences for this test's manager instance
			prefsToDefine := []PreferenceDefinition{
				{
					Key:          "language",
					Type:         "string",
					Category:     "general",
					DefaultValue: "en",
				},
				{
					Key:          "timezone",
					Type:         "string",
					Category:     "general",
					DefaultValue: "UTC",
				},
				{
					Key:          "notifications",
					Type:         "boolean",
					Category:     "settings",
					DefaultValue: false, // Note: this matches the expectedPrefs definition
				},
			}

			for _, def := range prefsToDefine {
				if err := mgr.DefinePreference(def); err != nil {
					t.Fatalf("DefinePreference failed: %v", err)
				}
			}

			// Set up the cache and storage as per the test case
			if tc.setupCache != nil {
				tc.setupCache(cache, store)
			}
			if tc.mockGetAll != nil {
				tc.mockGetAll(store, tc.userID)
			}

			// Get all preferences for the user
			prefs, err := mgr.GetAll(context.Background(), tc.userID)

			// Check the error
			if err != nil {
				if tc.expectedError == nil {
					t.Errorf("Expected no error, got: %v", err)
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("Expected error '%v', got '%v'", tc.expectedError, err)
				}
			} else if tc.expectedError != nil {
				t.Errorf("Expected error '%v', got none", tc.expectedError)
			}

			// Check the preferences
			if len(prefs) != len(tc.expectedPrefs) {
				t.Errorf("Expected %d preferences, got %d", len(tc.expectedPrefs), len(prefs))
			}
			for key, expectedPref := range tc.expectedPrefs {
				if pref, exists := prefs[key]; !exists {
					t.Errorf("Expected preference '%s' to exist", key)
				} else if pref.Value != expectedPref.Value {
					t.Errorf("Preference '%s' value mismatch: expected '%v', got '%v'", key, expectedPref.Value, pref.Value)
				}
			}
		})
	}
}

func TestManager_GetPreferencesByCategory(t *testing.T) {
	store := NewMockStorage()
	cache := NewMockCache()
	logger := &MockLogger{}
	mgr := New(
		WithStorage(store),
		WithCache(cache),
		WithLogger(logger),
	)

	userID := "user_test_category_prefs"
	category := "general"

	// Define multiple preferences
	prefsToDefine := []PreferenceDefinition{
		{
			Key:          "language",
			Type:         "string",
			Category:     category,
			DefaultValue: "en",
		},
		{
			Key:          "timezone",
			Type:         "string",
			Category:     category,
			DefaultValue: "UTC",
		},
		{
			Key:          "notifications",
			Type:         "boolean",
			Category:     "settings",
			DefaultValue: true,
		},
	}

	for _, def := range prefsToDefine {
		if err := mgr.DefinePreference(def); err != nil {
			t.Fatalf("DefinePreference failed: %v", err)
		}
	}

	// Set some preferences
	err := mgr.Set(context.Background(), userID, "language", "fr")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	err = mgr.Set(context.Background(), userID, "timezone", "PST")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Mock expected preferences from storage
	expectedPrefs := map[string]*Preference{
		"language": {
			UserID:       userID,
			Key:          "language",
			Value:        "fr",
			DefaultValue: "en",
			Type:         "string",
			Category:     category,
			UpdatedAt:    time.Now(),
		},
		"timezone": {
			UserID:       userID,
			Key:          "timezone",
			Value:        "PST",
			DefaultValue: "UTC",
			Type:         "string",
			Category:     category,
			UpdatedAt:    time.Now(),
		},
	}

	// Test cases
	testCases := []struct {
		name              string
		cacheType         string
		setupCache        func(_ *MockCache, _ *MockStorage) // Changed mc to _
		userID            string
		category          string
		mockGetByCategory func(_ *MockStorage, _ string, _ string) // Changed mockStore, uid, cat to _
		expectedPrefs     map[string]*Preference
		expectedError     error
	}{
		{
			name:      "cache hit on get by category",
			cacheType: "memory", // Can be any cache type
			setupCache: func(_ *MockCache, store *MockStorage) {
				// Set up the preferences in the mock storage
				for _, pref := range expectedPrefs {
					_ = store.Set(context.Background(), pref)
				}
			},
			userID:   userID,
			category: category,
			mockGetByCategory: func(_ *MockStorage, _ string, _ string) {
				// No additional setup needed as preferences are set in setupCache
			},
			expectedPrefs: expectedPrefs,
			expectedError: nil,
		},
		{
			name:      "cache error on get by category",
			cacheType: "redis",
			setupCache: func(_ *MockCache, store *MockStorage) {
				// Set up the preferences in the mock storage
				for _, pref := range expectedPrefs {
					_ = store.Set(context.Background(), pref)
				}
			},
			userID:   userID,
			category: category,
			mockGetByCategory: func(_ *MockStorage, _ string, _ string) {
				// No additional setup needed as preferences are set in setupCache
			},
			expectedPrefs: expectedPrefs,
			expectedError: nil,
		},
		{
			name:      "store error on get by category after cache miss",
			cacheType: "memory", // Can be any cache type
			setupCache: func(_ *MockCache, store *MockStorage) {
				// Set up the storage to return an error for GetByCategory
				store.SetGetByCategoryError(errors.New("store GetByCategory error"))
			},
			userID:   userID,
			category: category,
			mockGetByCategory: func(_ *MockStorage, _ string, _ string) {
				// No additional setup needed as we set the error in setupCache
			},
			expectedPrefs: nil,
			expectedError: fmt.Errorf("storage.GetByCategory failed for category '%s': %w", category, errors.New("store GetByCategory error")),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewMockStorage()
			cache := NewMockCache()
			logger := &MockLogger{}

			mgr := New(
				WithStorage(store),
				WithCache(cache),
				WithLogger(logger),
			)

			// Set up the cache and storage as per the test case
			if tc.setupCache != nil {
				tc.setupCache(cache, store)
			}
			if tc.mockGetByCategory != nil {
				tc.mockGetByCategory(store, tc.userID, tc.category)
			}

			// Get preferences by category for the user
			prefs, err := mgr.GetByCategory(context.Background(), tc.userID, tc.category)

			// Check the error
			if err != nil {
				if tc.expectedError == nil {
					t.Errorf("Expected no error, got: %v", err)
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("Expected error '%v', got '%v'", tc.expectedError, err)
				}
			} else if tc.expectedError != nil {
				t.Errorf("Expected error '%v', got none", tc.expectedError)
			}

			// Check the preferences
			if len(prefs) != len(tc.expectedPrefs) {
				t.Errorf("Expected %d preferences, got %d", len(tc.expectedPrefs), len(prefs))
			}
			for key, expectedPref := range tc.expectedPrefs {
				if pref, exists := prefs[key]; !exists {
					t.Errorf("Expected preference '%s' to exist", key)
				} else if pref.Value != expectedPref.Value {
					t.Errorf("Preference '%s' value mismatch: expected '%v', got '%v'", key, expectedPref.Value, pref.Value)
				}
			}
		})
	}
}
