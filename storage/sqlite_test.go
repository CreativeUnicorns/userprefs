package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CreativeUnicorns/userprefs"
)

func TestSQLiteStorage(t *testing.T) {
	dbPath := "test_preferences.db"
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
	userID := "user_sqlite"
	key := "theme"

	// Define preference definition
	def := userprefs.PreferenceDefinition{
		Key:          "theme",
		Type:         "enum",
		Category:     "appearance",
		DefaultValue: "light",
		AllowedValues: []interface{}{
			"light",
			"dark",
			"system",
		},
	}

	// Attempt to get undefined preference (should return ErrNotFound)
	_, err = storage.Get(ctx, userID, key)
	if err != userprefs.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}

	// Set preference
	pref := &userprefs.Preference{
		UserID:    userID,
		Key:       key,
		Value:     "dark",
		Type:      def.Type,
		Category:  def.Category,
		UpdatedAt: time.Now(),
	}

	err = storage.Set(ctx, pref)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get preference
	retrieved, err := storage.Get(ctx, userID, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.Value != "dark" {
		t.Errorf("Expected 'dark', got '%v'", retrieved.Value)
	}

	// Update preference
	pref.Value = "light"
	err = storage.Set(ctx, pref)
	if err != nil {
		t.Fatalf("Set (update) failed: %v", err)
	}

	// Get updated preference
	retrieved, err = storage.Get(ctx, userID, key)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if retrieved.Value != "light" {
		t.Errorf("Expected 'light', got '%v'", retrieved.Value)
	}

	// Get all preferences for user
	allPrefs, err := storage.GetAll(ctx, userID)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(allPrefs) != 1 {
		t.Errorf("Expected 1 preference, got %d", len(allPrefs))
	}

	// Get by category
	appearancePrefs, err := storage.GetByCategory(ctx, userID, "appearance")
	if err != nil {
		t.Fatalf("GetByCategory failed: %v", err)
	}
	if len(appearancePrefs) != 1 {
		t.Errorf("Expected 1 appearance preference, got %d", len(appearancePrefs))
	}

	// Delete preference
	err = storage.Delete(ctx, userID, key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Ensure deletion
	_, err = storage.Get(ctx, userID, key)
	if err != userprefs.ErrNotFound {
		t.Fatalf("Expected ErrNotFound after deletion, got: %v", err)
	}
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

	// The final value should be between 0 and 99
	val, ok := finalPref.Value.(float64) // SQLite returns numbers as float64
	if !ok {
		t.Fatalf("Expected float64 value, got %T", finalPref.Value)
	}
	if val < 0 || val > 99 {
		t.Errorf("Final volume out of expected range: %v", val)
	}
}
