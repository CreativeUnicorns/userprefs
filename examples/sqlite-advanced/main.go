// examples/sqlite-advanced/main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

// CommandPreferences represents specific command settings
type CommandPreferences struct {
	Enabled     bool              `json:"enabled"`
	Cooldown    int               `json:"cooldown_seconds"`
	Permissions map[string]bool   `json:"permissions"`
	Defaults    map[string]string `json:"defaults"`
}

// BotPreferences represents global bot settings
type BotPreferences struct {
	Language       string   `json:"language"`
	Prefix         string   `json:"prefix"`
	AllowedRoles   []string `json:"allowed_roles"`
	AllowedServers []string `json:"allowed_servers"`
}

func main() {
	// Initialize SQLite storage with in-memory cache
	store, err := storage.NewSQLiteStorage("sqlite-advanced.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize in-memory cache
	memCache := cache.NewMemoryCache()
	defer memCache.Close()

	// Create preference manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(memCache),
	)

	// Define complex command preferences
	commandPrefs := map[string]CommandPreferences{
		"gif": {
			Enabled:  true,
			Cooldown: 10,
			Permissions: map[string]bool{
				"create": true,
				"edit":   true,
				"delete": false,
			},
			Defaults: map[string]string{
				"format": "gif",
				"size":   "medium",
				"speed":  "1x",
			},
		},
		"convert": {
			Enabled:  true,
			Cooldown: 30,
			Permissions: map[string]bool{
				"create": true,
				"edit":   false,
				"delete": false,
			},
			Defaults: map[string]string{
				"format": "mp4",
				"fps":    "30",
				"scale":  "720p",
			},
		},
	}

	// Define preferences
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:      "bot_settings",
			Type:     "json",
			Category: "system",
			DefaultValue: BotPreferences{
				Language:       "en",
				Prefix:         "!",
				AllowedRoles:   []string{"admin", "moderator"},
				AllowedServers: []string{"*"},
			},
		},
	}

	// Add command preferences dynamically
	for cmdName, cmdPref := range commandPrefs {
		preferences = append(preferences, userprefs.PreferenceDefinition{
			Key:          "cmd_" + cmdName,
			Type:         "json",
			Category:     "commands",
			DefaultValue: cmdPref,
		})
	}

	ctx := context.Background()

	// Register all preferences
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	// Simulate different users and guilds
	userScenarios := []struct {
		userID  string
		guildID string
	}{
		{"user1", "guild1"},
		{"user2", "guild1"},
		{"user1", "guild2"},
	}

	// Test concurrent operations
	for _, scenario := range userScenarios {
		// Create composite key for user+guild specific settings
		userKey := scenario.userID + ":" + scenario.guildID

		// Simulate concurrent preference operations
		go func(userKey string) {
			// Set command preferences
			cmdPref := CommandPreferences{
				Enabled:  true,
				Cooldown: 5,
				Permissions: map[string]bool{
					"create": true,
					"edit":   true,
					"delete": true,
				},
				Defaults: map[string]string{
					"format": "webp",
					"size":   "large",
					"speed":  "2x",
				},
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			if err := mgr.Set(ctx, userKey, "cmd_gif", cmdPref); err != nil {
				log.Printf("Failed to set gif preferences for %s: %v", userKey, err)
				return
			}

			// Immediate read to test cache
			if pref, err := mgr.Get(ctx, userKey, "cmd_gif"); err != nil {
				log.Printf("Failed to get gif preferences for %s: %v", userKey, err)
			} else {
				var retrieved CommandPreferences
				data, _ := json.Marshal(pref.Value)
				if err := json.Unmarshal(data, &retrieved); err != nil {
					log.Printf("Failed to parse gif preferences for %s: %v", userKey, err)
					return
				}
				log.Printf("Retrieved gif preferences for %s: %+v", userKey, retrieved)
			}
		}(userKey)
	}

	// Demo bulk operations
	userKey := userScenarios[0].userID + ":" + userScenarios[0].guildID

	// Get all command preferences for a user
	if prefs, err := mgr.GetByCategory(ctx, userKey, "commands"); err != nil {
		log.Printf("Failed to get command preferences: %v", err)
	} else {
		log.Printf("\nAll command preferences for %s:", userKey)
		for key, pref := range prefs {
			data, _ := json.Marshal(pref.Value)
			log.Printf("  %s: %s", key, string(data))
		}
	}

	// Demonstrate cache performance
	log.Printf("\nDemonstrating cache performance:")
	start := time.Now()

	for i := 0; i < 1000; i++ {
		if _, err := mgr.Get(ctx, userKey, "cmd_gif"); err != nil {
			log.Printf("Cache test failed: %v", err)
			break
		}
	}

	log.Printf("1000 cached reads took: %v", time.Since(start))

	// Wait for goroutines to finish
	time.Sleep(time.Second * 5)

	// Final verification of settings
	botSettings, err := mgr.Get(ctx, userKey, "bot_settings")
	if err != nil {
		log.Printf("Failed to get bot settings: %v", err)
	} else {
		var settings BotPreferences
		data, _ := json.Marshal(botSettings.Value)
		if err := json.Unmarshal(data, &settings); err != nil {
			log.Printf("Failed to parse bot settings: %v", err)
		} else {
			log.Printf("\nFinal bot settings for %s: %+v", userKey, settings)
		}
	}
}
