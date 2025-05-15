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

var (
	appStorage userprefs.Storage
	appCache   userprefs.Cache
)

func main() {
	// Initialize PostgreSQL storage (used for the main part of the example)
	pgStore, err := storage.NewPostgresStorage("postgres://userprefs_user:userprefs_password@localhost:5432/userprefs_db?sslmode=disable")
	if err != nil {
		log.Printf("Failed to initialize PostgreSQL storage: %v. Main demo may use MemoryStorage.", err)
		// Allow example to continue with memory storage for other parts if PG fails
	} else {
		appStorage = pgStore
		defer func() {
			if err := pgStore.Close(); err != nil {
				log.Printf("Failed to close storage: %v", err)
			}
		}()
	}

	// Initialize Redis cache (used for the main part of the example)
	actualRedisCache, err := cache.NewRedisCache("localhost:6379", "", 0)
	if err != nil {
		log.Printf("Redis cache not available: %v. Main demo may use MemoryCache or no cache.", err)
		// actualRedisCache will be nil, appCache will not be set here
	} else {
		appCache = actualRedisCache
	}
	defer func() {
		if actualRedisCache != nil {
			if err := actualRedisCache.Close(); err != nil {
				log.Printf("Failed to close Redis cache: %v", err)
			}
		}
	}()

	// Create preference manager with storage and optional cache for the main demo
	mgrOptions := []userprefs.Option{}
	if appStorage != nil {
		mgrOptions = append(mgrOptions, userprefs.WithStorage(appStorage))
	} else {
		log.Println("Warning: PostgreSQL storage not available. Using MemoryStorage for main demo.")
		memStore := storage.NewMemoryStorage()
		appStorage = memStore // For a non-nil defer Close on store if pgStore was nil
		mgrOptions = append(mgrOptions, userprefs.WithStorage(memStore))
	}

	if appCache != nil {
		mgrOptions = append(mgrOptions, userprefs.WithCache(appCache))
	} else {
		log.Println("Warning: Redis cache not available. Using MemoryCache for main demo if storage is initialized.")
		if appStorage != nil { // Only add memory cache if we have some storage
			memCache := cache.NewMemoryCache()
			log.Println("Using in-memory cache for this demo.")
			appCache = memCache // For a non-nil defer Close on redisCache if actualRedisCache was nil
			mgrOptions = append(mgrOptions, userprefs.WithCache(memCache))
		}
	}

	mgr := userprefs.New(mgrOptions...)

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

	log.Println("\n--- Demonstrating Default Value Fallback and Cache Error Handling (using MemoryStorage/MemoryCache) ---")
	demonstrateDefaultAndCacheErrorHandling(ctx)
}

// demonstrateDefaultAndCacheErrorHandling demonstrates default value fallbacks
// and discusses scenarios for cache error handling.
func demonstrateDefaultAndCacheErrorHandling(ctx context.Context) {
	log.Println("\nInitializing Manager with MemoryStorage and MemoryCache for focused demo...")
	memStorage := storage.NewMemoryStorage()
	log.Println("Using in-memory storage for this demo.")
	memCacheInstance := cache.NewMemoryCache()

	// The manager will use its internal default logger
	mgr := userprefs.New(
		userprefs.WithStorage(memStorage),
		userprefs.WithCache(memCacheInstance),
	)

	// Define preferences specifically for this demonstration
	soundProfileDef := userprefs.PreferenceDefinition{
		Key:          "sound_profile_demo",
		Type:         "string",
		Category:     "audio_demo",
		DefaultValue: "stereo_default",
	}
	uiModeDef := userprefs.PreferenceDefinition{
		Key:          "ui_mode_demo",
		Type:         "string",
		Category:     "appearance_demo",
		DefaultValue: "light_default",
	}

	// It's important to define preferences before using them
	if err := mgr.DefinePreference(soundProfileDef); err != nil {
		log.Fatalf("Demo: Failed to define preference '%s': %v", soundProfileDef.Key, err)
	}
	if err := mgr.DefinePreference(uiModeDef); err != nil {
		log.Fatalf("Demo: Failed to define preference '%s': %v", uiModeDef.Key, err)
	}

	userID := "demo_user_for_defaults"

	log.Println("\n--- 1. Default Value Fallback (Preference Not Set) ---")
	log.Printf("Attempting to get '%s' for user '%s' (preference has not been set by user yet)...", soundProfileDef.Key, userID)
	pref, err := mgr.Get(ctx, userID, soundProfileDef.Key)
	if err != nil {
		// If Get returns an error, it's a genuine problem (not a default fallback).
		// However, our current Get implementation returns default with nil error for not found,
		// and default with cacheErr for cache issues. So, a non-nil err here is unexpected for this flow.
		log.Printf("Demo: Error getting '%s': %v", soundProfileDef.Key, err)
	} else {
		// Check if the returned value is the default. The Get method ensures DefaultValue field is populated.
		isActuallyDefault := pref.Value == pref.DefaultValue
		log.Printf("Demo: Got '%s': Value='%v', DefaultValueInPref='%v', IsActuallyDefault=%t. Expected DefinitionDefault: '%s'", pref.Key, pref.Value, pref.DefaultValue, isActuallyDefault, soundProfileDef.DefaultValue)
		if isActuallyDefault && pref.Value == soundProfileDef.DefaultValue {
			log.Printf("Demo: SUCCESS - Correctly received the default value for an unset preference.")
		} else {
			log.Printf("Demo: FAILED - Did not receive the expected default value or behavior. Got Value: %v, IsActuallyDefault: %t", pref.Value, isActuallyDefault)
		}
	}

	log.Println("\n--- 2. Cache Error Fallback Demonstration (Conceptual) ---")
	customUIModeValue := "dark_custom_theme"
	log.Printf("Setting '%s' for user '%s' to '%s'", uiModeDef.Key, userID, customUIModeValue)
	if err := mgr.Set(ctx, userID, uiModeDef.Key, customUIModeValue); err != nil {
		log.Fatalf("Demo: Failed to set '%s': %v", uiModeDef.Key, err)
	}
	pref, err = mgr.Get(ctx, userID, uiModeDef.Key)
	if err != nil || pref.Value != customUIModeValue {
		log.Fatalf("Demo: Failed to get '%s' after setting, or value mismatch: Error='%v', Value='%v'", uiModeDef.Key, err, pref.Value)
	}
	isActuallyDefaultAfterSet := pref.Value == pref.DefaultValue // Should be false now
	log.Printf("Demo: Successfully set and retrieved '%s', value: '%v', IsActuallyDefault=%t", uiModeDef.Key, pref.Value, isActuallyDefaultAfterSet)

	log.Println("\nSimulating scenario where cache data for a key is corrupted (e.g., bad JSON)...")
	log.Println("   (This is conceptually demonstrated as direct cache manipulation is complex in an example).")
	log.Println("   Unit tests for Manager cover cache error scenarios (e.g., ErrSerialization) with a mock cache.")
	log.Println("   Expected behavior: Manager logs cache error, attempts to fetch from storage.")
	log.Println("   If storage has the item, it's returned and cache is repopulated.")
	log.Println("   If storage *also* fails or misses for that key, *then* the DefaultValue for the preference is returned.")

	log.Printf("Assuming cache for '%s' (value '%s') is now corrupted. Attempting Get operation...", uiModeDef.Key, customUIModeValue)
	// In a real corruption scenario where getFromCache returns ErrSerialization for uiModeDef.Key:
	// 1. Manager logs the cache error.
	// 2. Manager attempts to get uiModeDef.Key from storage.
	// 3. Storage has 'dark_custom_theme', so it returns that.
	// 4. Cache is repopulated with 'dark_custom_theme'.
	// So, user still gets 'dark_custom_theme'.
	log.Println("   If storage had ALSO failed for uiModeDef.Key, THEN user would get DefaultValue: '" + uiModeDef.DefaultValue.(string) + "'")

	// To unequivocally show default fallback from a 'bad' state, let's use a new key that will only ever have a default.
	neverSetPrefKey := "never_set_pref_demo"
	neverSetPrefDef := userprefs.PreferenceDefinition{
		Key: neverSetPrefKey, Type: "string", DefaultValue: "default_for_never_set", Category: "demo_extra",
	}
	if err := mgr.DefinePreference(neverSetPrefDef); err != nil { log.Fatalf("Failed to define %s: %v", neverSetPrefKey, err); }

	log.Printf("Fetching '%s' for user '%s' (never set, simulating path that leads to default)...", neverSetPrefKey, userID)
	pref, err = mgr.Get(ctx, userID, neverSetPrefKey)
	if err != nil {
		// Similar to the first Get, a non-nil err is unexpected for this flow.
		log.Printf("Demo: Error getting '%s': %v", neverSetPrefKey, err)
	} else {
		isActuallyDefaultNeverSet := pref.Value == pref.DefaultValue
		log.Printf("Demo: Got '%s': Value='%v', DefaultValueInPref='%v', IsActuallyDefault=%t. Expected DefinitionDefault: '%s'", pref.Key, pref.Value, pref.DefaultValue, isActuallyDefaultNeverSet, neverSetPrefDef.DefaultValue)
		if isActuallyDefaultNeverSet && pref.Value == neverSetPrefDef.DefaultValue {
			log.Printf("Demo: SUCCESS - Fallback to default value demonstrated for '%s'.", neverSetPrefKey)
		}
	}
	log.Println("\nAdvanced example demonstration finished.")
}
