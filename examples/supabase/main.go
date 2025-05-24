// examples/supabase/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

// UserProfile represents a user's profile preferences
type UserProfile struct {
	Theme         string `json:"theme"`
	Language      string `json:"language"`
	Timezone      string `json:"timezone"`
	EmailSettings struct {
		Marketing bool `json:"marketing"`
		Security  bool `json:"security"`
		Updates   bool `json:"updates"`
	} `json:"email_settings"`
	UIPreferences struct {
		SidebarCollapsed bool   `json:"sidebar_collapsed"`
		DensityMode      string `json:"density_mode"`
		FontSize         int    `json:"font_size"`
	} `json:"ui_preferences"`
}

// ProjectSettings represents project-specific configuration
type ProjectSettings struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsPublic    bool     `json:"is_public"`
	Tags        []string `json:"tags"`
	Settings    struct {
		AutoSave         bool `json:"auto_save"`
		AutoSaveInterval int  `json:"auto_save_interval"`
		MaxFileSize      int  `json:"max_file_size"`
	} `json:"settings"`
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
		log.Println("Make sure to set environment variables or copy .env.example to .env")
	}

	// Get required environment variables
	supabaseURL := os.Getenv("SUPABASE_URL")
	dbURL := os.Getenv("SUPABASE_DB_URL")

	if supabaseURL == "" || dbURL == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_DB_URL environment variables are required")
	}

	log.Printf("Connecting to Supabase at: %s", supabaseURL)

	// Initialize Supabase PostgreSQL storage
	store, err := storage.NewPostgresStorage(storage.WithPostgresDSN(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to Supabase database: %v", err)
	}
	defer store.Close()

	// Initialize cache if Redis is available
	var cacheInstance userprefs.Cache
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
		redisPassword := os.Getenv("REDIS_PASSWORD")

		redisCache, err := cache.NewRedisCache(
			cache.WithRedisAddress(redisURL),
			cache.WithRedisPassword(redisPassword),
			cache.WithRedisDB(redisDB),
		)
		if err != nil {
			log.Printf("Redis cache not available, using memory cache: %v", err)
			cacheInstance = cache.NewMemoryCache()
		} else {
			cacheInstance = redisCache
			log.Println("Using Redis cache for better performance")
		}
	} else {
		cacheInstance = cache.NewMemoryCache()
		log.Println("Using in-memory cache")
	}
	defer cacheInstance.Close()

	// Create preference manager with Supabase storage and cache
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(cacheInstance),
	)

	// Define comprehensive preference definitions
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:      "user_profile",
			Type:     "json",
			Category: "user",
			DefaultValue: UserProfile{
				Theme:    "dark",
				Language: "en",
				Timezone: "UTC",
				EmailSettings: struct {
					Marketing bool `json:"marketing"`
					Security  bool `json:"security"`
					Updates   bool `json:"updates"`
				}{
					Marketing: false,
					Security:  true,
					Updates:   true,
				},
				UIPreferences: struct {
					SidebarCollapsed bool   `json:"sidebar_collapsed"`
					DensityMode      string `json:"density_mode"`
					FontSize         int    `json:"font_size"`
				}{
					SidebarCollapsed: false,
					DensityMode:      "comfortable",
					FontSize:         14,
				},
			},
		},
		{
			Key:      "project_settings",
			Type:     "json",
			Category: "project",
			DefaultValue: ProjectSettings{
				Name:        "New Project",
				Description: "",
				IsPublic:    false,
				Tags:        []string{},
				Settings: struct {
					AutoSave         bool `json:"auto_save"`
					AutoSaveInterval int  `json:"auto_save_interval"`
					MaxFileSize      int  `json:"max_file_size"`
				}{
					AutoSave:         true,
					AutoSaveInterval: 30,
					MaxFileSize:      10485760, // 10MB
				},
			},
		},
		{
			Key:          "notification_level",
			Type:         "enum",
			Category:     "notifications",
			DefaultValue: "normal",
			AllowedValues: []interface{}{
				"silent",
				"normal",
				"verbose",
			},
		},
		{
			Key:          "beta_features_enabled",
			Type:         "boolean",
			Category:     "experimental",
			DefaultValue: false,
		},
		{
			Key:          "max_concurrent_uploads",
			Type:         "number",
			Category:     "performance",
			DefaultValue: 5,
		},
	}

	ctx := context.Background()

	// Register all preferences
	log.Println("Registering preference definitions...")
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	// Simulate real-world usage scenarios
	log.Println("\n=== Supabase User Preferences Demo ===")

	// Scenario 1: New user with default preferences
	userID1 := "user_" + fmt.Sprintf("%d", time.Now().Unix())
	log.Printf("\nScenario 1: New user (%s) - Getting default preferences", userID1)

	profile, err := mgr.Get(ctx, userID1, "user_profile")
	if err != nil {
		log.Fatalf("Failed to get user profile: %v", err)
	}

	var userProfile UserProfile
	profileData, _ := json.Marshal(profile.Value)
	json.Unmarshal(profileData, &userProfile)
	log.Printf("Default theme: %s, Language: %s", userProfile.Theme, userProfile.Language)

	// Scenario 2: User customizes their preferences
	log.Printf("\nScenario 2: User (%s) customizes preferences", userID1)

	// Update user profile
	userProfile.Theme = "light"
	userProfile.Language = "es"
	userProfile.EmailSettings.Marketing = true
	userProfile.UIPreferences.FontSize = 16

	err = mgr.Set(ctx, userID1, "user_profile", userProfile)
	if err != nil {
		log.Fatalf("Failed to update user profile: %v", err)
	}

	// Enable beta features
	err = mgr.Set(ctx, userID1, "beta_features_enabled", true)
	if err != nil {
		log.Fatalf("Failed to enable beta features: %v", err)
	}

	// Set notification level
	err = mgr.Set(ctx, userID1, "notification_level", "verbose")
	if err != nil {
		log.Fatalf("Failed to set notification level: %v", err)
	}

	log.Println("✅ User preferences updated successfully")

	// Scenario 3: Multiple users with different preferences
	log.Println("\nScenario 3: Multiple users with different preferences")

	users := []string{
		"alice_dev",
		"bob_designer",
		"charlie_manager",
	}

	for i, userID := range users {
		// Customize project settings for each user
		var projectSettings ProjectSettings
		projectData, _ := json.Marshal(preferences[1].DefaultValue)
		json.Unmarshal(projectData, &projectSettings)

		projectSettings.Name = fmt.Sprintf("Project_%s", userID)
		projectSettings.Description = fmt.Sprintf("Project managed by %s", userID)
		projectSettings.IsPublic = i%2 == 0 // Every other user makes projects public
		projectSettings.Tags = []string{fmt.Sprintf("team_%d", i+1), "supabase-demo"}

		err = mgr.Set(ctx, userID, "project_settings", projectSettings)
		if err != nil {
			log.Printf("Failed to set project settings for %s: %v", userID, err)
			continue
		}

		// Set different upload limits based on user role
		uploadLimit := 3 + i*2 // 3, 5, 7
		err = mgr.Set(ctx, userID, "max_concurrent_uploads", uploadLimit)
		if err != nil {
			log.Printf("Failed to set upload limit for %s: %v", userID, err)
			continue
		}

		log.Printf("✅ Configured preferences for %s (upload limit: %d)", userID, uploadLimit)
	}

	// Scenario 4: Bulk operations and category-based retrieval
	log.Println("\nScenario 4: Bulk operations and analytics")

	// Get all preferences for the first user
	allPrefs, err := mgr.GetAll(ctx, userID1)
	if err != nil {
		log.Fatalf("Failed to get all preferences: %v", err)
	}

	log.Printf("User %s has %d preferences configured:", userID1, len(allPrefs))
	for key, pref := range allPrefs {
		log.Printf("  %s [%s]: %v", key, pref.Category, getValueSummary(pref.Value))
	}

	// Get all user-category preferences across all demo users
	log.Println("\nUser category preferences across all demo users:")
	allUsers := append([]string{userID1}, users...)
	for _, userID := range allUsers {
		userPrefs, err := mgr.GetByCategory(ctx, userID, "user")
		if err != nil {
			log.Printf("Failed to get user category prefs for %s: %v", userID, err)
			continue
		}

		if profile, exists := userPrefs["user_profile"]; exists {
			var up UserProfile
			profileData, _ := json.Marshal(profile.Value)
			json.Unmarshal(profileData, &up)
			log.Printf("  %s: theme=%s, lang=%s, font_size=%d",
				userID, up.Theme, up.Language, up.UIPreferences.FontSize)
		}
	}

	// Scenario 5: Performance demonstration with caching
	log.Println("\nScenario 5: Performance demonstration")

	// Measure cache performance
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := mgr.Get(ctx, userID1, "user_profile")
		if err != nil {
			log.Printf("Error in performance test: %v", err)
			break
		}
	}
	duration := time.Since(start)
	log.Printf("100 cached reads took: %v (avg: %v per read)", duration, duration/100)

	// Scenario 6: Real-time updates simulation
	log.Println("\nScenario 6: Real-time preference updates")

	// Simulate concurrent updates from different sessions
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(sessionID int) {
			defer func() { done <- true }()

			sessionUserID := fmt.Sprintf("concurrent_user_%d", sessionID)

			// Each session updates different preferences
			switch sessionID {
			case 0:
				err := mgr.Set(ctx, sessionUserID, "notification_level", "silent")
				if err != nil {
					log.Printf("Session %d failed to update notification: %v", sessionID, err)
				}
			case 1:
				err := mgr.Set(ctx, sessionUserID, "beta_features_enabled", true)
				if err != nil {
					log.Printf("Session %d failed to update beta features: %v", sessionID, err)
				}
			case 2:
				err := mgr.Set(ctx, sessionUserID, "max_concurrent_uploads", 10)
				if err != nil {
					log.Printf("Session %d failed to update upload limit: %v", sessionID, err)
				}
			}

			log.Printf("✅ Session %d completed preference update", sessionID)
		}(i)
	}

	// Wait for all concurrent operations
	for i := 0; i < 3; i++ {
		<-done
	}

	log.Println("✅ All concurrent updates completed")

	// Final summary
	log.Println("\n=== Demo Summary ===")
	log.Printf("✅ Successfully demonstrated Supabase integration with userprefs")
	log.Printf("✅ Used environment variables for secure configuration")
	log.Printf("✅ Tested complex JSON preferences with nested structures")
	log.Printf("✅ Demonstrated caching for improved performance")
	log.Printf("✅ Showed concurrent operations and real-time updates")
	log.Printf("✅ Connected to Supabase PostgreSQL: %s", supabaseURL)

	if os.Getenv("REDIS_URL") != "" {
		log.Printf("✅ Used Redis cache for optimal performance")
	} else {
		log.Printf("ℹ️  Used in-memory cache (set REDIS_URL for Redis caching)")
	}
}

// getValueSummary returns a brief summary of a preference value for logging
func getValueSummary(value interface{}) string {
	switch v := value.(type) {
	case string:
		if len(v) > 30 {
			return v[:27] + "..."
		}
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		// For complex objects, show type
		jsonData, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%T", v)
		}
		if len(jsonData) > 50 {
			return fmt.Sprintf("%T{...}", v)
		}
		return string(jsonData)
	}
}
