// examples/advanced/main.go
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

// Custom preference types
type VideoSettings struct {
	Resolution string `json:"resolution"`
	Framerate  int    `json:"framerate"`
	Codec      string `json:"codec"`
}

type NotificationSettings struct {
	Enabled  bool     `json:"enabled"`
	Channels []string `json:"channels"`
	Times    []string `json:"times"`
}

func main() {
	// Initialize PostgreSQL storage
	store, err := storage.NewPostgresStorage("postgres://localhost:5432/myapp?sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Failed to close storage: %v", err)
		}
	}()

	// Initialize Redis cache
	redisCache, err := cache.NewRedisCache("localhost:6379", "", 0)
	if err != nil {
		log.Printf("Redis cache not available: %v", err)
		redisCache = nil
	}
	defer func() {
		if redisCache != nil {
			if err := redisCache.Close(); err != nil {
				log.Printf("Failed to close Redis cache: %v", err)
			}
		}
	}()

	// Create preference manager with storage and optional cache
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(redisCache),
	)

	// Define complex preferences
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:      "video_settings",
			Type:     "json",
			Category: "advanced",
			DefaultValue: VideoSettings{
				Resolution: "1080p",
				Framerate:  30,
				Codec:      "h264",
			},
		},
		{
			Key:      "notification_settings",
			Type:     "json",
			Category: "notifications",
			DefaultValue: NotificationSettings{
				Enabled:  true,
				Channels: []string{"dm"},
				Times:    []string{"09:00", "18:00"},
			},
		},
		{
			Key:          "theme",
			Type:         "enum",
			Category:     "appearance",
			DefaultValue: "dark",
			AllowedValues: []interface{}{
				"light",
				"dark",
				"system",
			},
		},
	}

	ctx := context.Background()

	// Register preferences
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	// Simulate multiple users
	users := []string{"user1", "user2", "user3"}

	for _, userID := range users {
		// Set custom video settings for each user
		videoSettings := VideoSettings{
			Resolution: "1080p",
			Framerate:  60,
			Codec:      "h265",
		}

		if err := mgr.Set(ctx, userID, "video_settings", videoSettings); err != nil {
			log.Printf("Failed to set video settings for user %s: %v", userID, err)
			continue
		}

		// Demonstrate getting complex preferences
		pref, err := mgr.Get(ctx, userID, "video_settings")
		if err != nil {
			log.Printf("Failed to get video settings for user %s: %v", userID, err)
			continue
		}

		// Convert interface{} to VideoSettings
		settings := VideoSettings{}
		data, _ := json.Marshal(pref.Value)
		if err := json.Unmarshal(data, &settings); err != nil {
			log.Printf("Failed to parse video settings for user %s: %v", userID, err)
			continue
		}

		log.Printf("User %s video settings:", userID)
		log.Printf("  Resolution: %s", settings.Resolution)
		log.Printf("  Framerate: %d", settings.Framerate)
		log.Printf("  Codec: %s", settings.Codec)
	}

	// Demonstrate getting all preferences for a user
	userID := users[0]
	allPrefs, err := mgr.GetAll(ctx, userID)
	if err != nil {
		log.Fatalf("Failed to get all preferences: %v", err)
	}

	log.Printf("\nAll preferences for user %s:", userID)
	for key, pref := range allPrefs {
		data, _ := json.Marshal(pref.Value)
		log.Printf("  %s: %s", key, string(data))
	}

	// Demonstrate concurrent access
	for i := 0; i < 5; i++ {
		go func(userID string) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			if _, err := mgr.Get(ctx, userID, "theme"); err != nil {
				log.Printf("Concurrent get failed: %v", err)
			}
		}(userID)
	}

	// Wait a moment for concurrent operations
	time.Sleep(time.Second * 2)
}
