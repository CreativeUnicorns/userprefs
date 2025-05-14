package main

import (
	"context"
	"fmt"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	// Initialize storage
	store, _ := storage.NewSQLiteStorage("prefs.db")
	defer store.Close()

	// Create manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
	)

	// Define preferences
	mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "theme",
		Type:         "enum",
		Category:     "appearance",
		DefaultValue: "dark",
		AllowedValues: []interface{}{
			"light", "dark", "system",
		},
	})

	// Set preference
	ctx := context.Background()
	mgr.Set(ctx, "user123", "theme", "light")

	// Get preference
	pref, _ := mgr.Get(ctx, "user123", "theme")
	fmt.Printf("Theme: %v\n", pref.Value)
}
