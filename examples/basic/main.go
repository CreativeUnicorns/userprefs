// examples/basic/main.go
package main

import (
	"context"
	"log"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	// Initialize storage with SQLite for simplicity
	store, err := storage.NewSQLiteStorage("preferences.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create preference manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
	)

	// Define some preferences
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:          "output_format",
			Type:         userprefs.StringType,
			DefaultValue: "gif",
			Category:     "media",
			AllowedValues: []interface{}{
				"gif",
				"mp4",
				"webp",
			},
		},
		{
			Key:          "auto_convert",
			Type:         userprefs.BoolType,
			DefaultValue: false,
			Category:     "media",
		},
		{
			Key:          "max_duration",
			Type:         userprefs.IntType,
			DefaultValue: 30,
			Category:     "media",
		},
	}

	// Register preferences
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	ctx := context.Background()

	// Simulate user interactions
	userID := "user123"

	// Get preference with default value
	pref, err := mgr.Get(ctx, userID, "output_format")
	if err != nil {
		log.Fatalf("Failed to get preference: %v", err)
	}
	log.Printf("Default output format: %v", pref.Value)

	// Set some preferences
	err = mgr.Set(ctx, userID, "output_format", "mp4")
	if err != nil {
		log.Fatalf("Failed to set preference: %v", err)
	}

	err = mgr.Set(ctx, userID, "auto_convert", true)
	if err != nil {
		log.Fatalf("Failed to set preference: %v", err)
	}

	// Get all preferences for category
	prefs, err := mgr.GetByCategory(ctx, userID, "media")
	if err != nil {
		log.Fatalf("Failed to get preferences: %v", err)
	}

	log.Printf("All media preferences:")
	for key, pref := range prefs {
		log.Printf("  %s: %v", key, pref.Value)
	}
}
