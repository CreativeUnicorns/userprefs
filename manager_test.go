package userprefs

// import (
// 	"context"
// 	"testing"
// )

// func TestManager_DefinePreference(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	def := PreferenceDefinition{
// 		Key:          "theme",
// 		Type:         "enum",
// 		Category:     "appearance",
// 		DefaultValue: "dark",
// 		AllowedValues: []interface{}{
// 			"light",
// 			"dark",
// 			"system",
// 		},
// 	}

// 	err := mgr.DefinePreference(def)
// 	if err != nil {
// 		t.Fatalf("DefinePreference failed: %v", err)
// 	}

// 	// Attempt to define the same preference again
// 	err = mgr.DefinePreference(def)
// 	if err != nil {
// 		t.Fatalf("DefinePreference failed on redefining: %v", err)
// 	}

// 	// Define a preference with invalid type
// 	invalidDef := PreferenceDefinition{
// 		Key:  "invalid_pref",
// 		Type: "unsupported",
// 	}
// 	err = mgr.DefinePreference(invalidDef)
// 	if err == nil || err.Error() != "invalid preference type: unsupported" {
// 		t.Fatalf("Expected ErrInvalidType, got: %v", err)
// 	}
// }

// func TestManager_GetSet(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	def := PreferenceDefinition{
// 		Key:          "notifications",
// 		Type:         "boolean",
// 		Category:     "settings",
// 		DefaultValue: false,
// 	}

// 	err := mgr.DefinePreference(def)
// 	if err != nil {
// 		t.Fatalf("DefinePreference failed: %v", err)
// 	}

// 	userID := "user1"

// 	// Get preference when not set (should return default)
// 	pref, err := mgr.Get(context.Background(), userID, "notifications")
// 	if err != nil {
// 		t.Fatalf("Get failed: %v", err)
// 	}
// 	if pref.Value != false {
// 		t.Errorf("Expected default value false, got %v", pref.Value)
// 	}

// 	// Set preference
// 	err = mgr.Set(context.Background(), userID, "notifications", true)
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}

// 	// Get preference after setting
// 	pref, err = mgr.Get(context.Background(), userID, "notifications")
// 	if err != nil {
// 		t.Fatalf("Get failed: %v", err)
// 	}
// 	if pref.Value != true {
// 		t.Errorf("Expected value true, got %v", pref.Value)
// 	}

// 	// Ensure it's cached
// 	if _, exists := cache.data["pref:user1:notifications"]; !exists {
// 		t.Errorf("Preference not cached")
// 	}

// 	// Get from cache
// 	pref, err = mgr.Get(context.Background(), userID, "notifications")
// 	if err != nil {
// 		t.Fatalf("Get from cache failed: %v", err)
// 	}
// 	if pref.Value != true {
// 		t.Errorf("Expected cached value true, got %v", pref.Value)
// 	}
// }

// func TestManager_Delete(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	def := PreferenceDefinition{
// 		Key:          "volume",
// 		Type:         "number",
// 		Category:     "audio",
// 		DefaultValue: 50,
// 	}

// 	err := mgr.DefinePreference(def)
// 	if err != nil {
// 		t.Fatalf("DefinePreference failed: %v", err)
// 	}

// 	userID := "user2"

// 	// Set preference
// 	err = mgr.Set(context.Background(), userID, "volume", 75)
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}

// 	// Delete preference
// 	err = mgr.Delete(context.Background(), userID, "volume")
// 	if err != nil {
// 		t.Fatalf("Delete failed: %v", err)
// 	}

// 	// Get preference after deletion (should return default)
// 	pref, err := mgr.Get(context.Background(), userID, "volume")
// 	if err != nil {
// 		t.Fatalf("Get failed: %v", err)
// 	}
// 	if pref.Value != 50 {
// 		t.Errorf("Expected default value 50, got %v", pref.Value)
// 	}

// 	// Ensure cache is cleared
// 	if _, exists := cache.data["pref:user2:volume"]; exists {
// 		t.Errorf("Preference not deleted from cache")
// 	}

// 	// Attempt to delete non-existing preference
// 	err = mgr.Delete(context.Background(), userID, "nonexistent")
// 	if err != ErrNotFound {
// 		t.Errorf("Expected ErrNotFound, got %v", err)
// 	}
// }

// func TestManager_GetByCategory(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	// Define multiple preferences
// 	prefs := []PreferenceDefinition{
// 		{
// 			Key:          "theme",
// 			Type:         "enum",
// 			Category:     "appearance",
// 			DefaultValue: "light",
// 			AllowedValues: []interface{}{
// 				"light",
// 				"dark",
// 			},
// 		},
// 		{
// 			Key:          "font_size",
// 			Type:         "number",
// 			Category:     "appearance",
// 			DefaultValue: 12,
// 		},
// 		{
// 			Key:          "notifications",
// 			Type:         "boolean",
// 			Category:     "settings",
// 			DefaultValue: true,
// 		},
// 	}

// 	for _, def := range prefs {
// 		if err := mgr.DefinePreference(def); err != nil {
// 			t.Fatalf("DefinePreference failed: %v", err)
// 		}
// 	}

// 	userID := "user3"

// 	// Set some preferences
// 	err := mgr.Set(context.Background(), userID, "theme", "dark")
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}
// 	err = mgr.Set(context.Background(), userID, "font_size", 14)
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}

// 	// Get preferences by category
// 	appearancePrefs, err := mgr.GetByCategory(context.Background(), userID, "appearance")
// 	if err != nil {
// 		t.Fatalf("GetByCategory failed: %v", err)
// 	}

// 	if len(appearancePrefs) != 2 {
// 		t.Errorf("Expected 2 appearance preferences, got %d", len(appearancePrefs))
// 	}

// 	if appearancePrefs["theme"].Value != "dark" {
// 		t.Errorf("Expected theme 'dark', got %v", appearancePrefs["theme"].Value)
// 	}
// 	if appearancePrefs["font_size"].Value != 14 {
// 		t.Errorf("Expected font_size 14, got %v", appearancePrefs["font_size"].Value)
// 	}

// 	// Get preferences by non-existing category
// 	_, err = mgr.GetByCategory(context.Background(), userID, "nonexistent")
// 	if err != ErrNotFound {
// 		t.Errorf("Expected ErrNotFound for nonexistent category, got %v", err)
// 	}
// }

// func TestManager_GetAll(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	// Define multiple preferences
// 	prefs := []PreferenceDefinition{
// 		{
// 			Key:          "language",
// 			Type:         "string",
// 			Category:     "general",
// 			DefaultValue: "en",
// 		},
// 		{
// 			Key:          "timezone",
// 			Type:         "string",
// 			Category:     "general",
// 			DefaultValue: "UTC",
// 		},
// 	}

// 	for _, def := range prefs {
// 		if err := mgr.DefinePreference(def); err != nil {
// 			t.Fatalf("DefinePreference failed: %v", err)
// 		}
// 	}

// 	userID := "user4"

// 	// Set some preferences
// 	err := mgr.Set(context.Background(), userID, "language", "fr")
// 	if err != nil {
// 		t.Fatalf("Set failed: %v", err)
// 	}

// 	// Get all preferences
// 	allPrefs, err := mgr.GetAll(context.Background(), userID)
// 	if err != nil {
// 		t.Fatalf("GetAll failed: %v", err)
// 	}

// 	// Ensure all defined preferences are returned
// 	if len(allPrefs) != 2 {
// 		t.Errorf("Expected 2 preferences, got %d", len(allPrefs))
// 	}

// 	// Validate each preference
// 	if pref, exists := allPrefs["language"]; !exists || pref.Value != "fr" {
// 		t.Errorf("Expected language 'fr', got %v", pref.Value)
// 	}
// 	if pref, exists := allPrefs["timezone"]; !exists || pref.Value != "UTC" {
// 		t.Errorf("Expected timezone 'UTC', got %v", pref.Value)
// 	}
// }

// func TestManager_Concurrency(t *testing.T) {
// 	store := NewMockStorage()
// 	cache := NewMockCache()
// 	logger := &MockLogger{}
// 	mgr := &Manager{
// 		config: &Config{
// 			storage:     store,
// 			cache:       cache,
// 			logger:      logger,
// 			definitions: make(map[string]PreferenceDefinition),
// 		},
// 	}

// 	def := PreferenceDefinition{
// 		Key:          "volume",
// 		Type:         "number",
// 		Category:     "audio",
// 		DefaultValue: 50,
// 	}

// 	err := mgr.DefinePreference(def)
// 	if err != nil {
// 		t.Fatalf("DefinePreference failed: %v", err)
// 	}

// 	userID := "user5"
// 	done := make(chan bool)

// 	// Concurrently set preferences
// 	for i := 0; i < 100; i++ {
// 		go func(val int) {
// 			if err := mgr.Set(context.Background(), userID, "volume", val); err != nil {
// 				t.Errorf("Set failed: %v", err)
// 			}
// 			done <- true
// 		}(i)
// 	}

// 	// Wait for all goroutines to finish
// 	for i := 0; i < 100; i++ {
// 		<-done
// 	}

// 	// Get the final value
// 	pref, err := mgr.Get(context.Background(), userID, "volume")
// 	if err != nil {
// 		t.Fatalf("Get failed: %v", err)
// 	}

// 	// The final value should be between 0 and 99
// 	switch v := pref.Value.(type) {
// 	case int:
// 		if v < 0 || v > 99 {
// 			t.Errorf("Final volume out of expected range: %v", v)
// 		}
// 	case float64:
// 		if int(v) < 0 || int(v) > 99 {
// 			t.Errorf("Final volume out of expected range: %v", v)
// 		}
// 	default:
// 		t.Errorf("Unexpected type for volume: %T", v)
// 	}
// }
