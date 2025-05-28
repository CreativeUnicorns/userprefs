// examples/webapp/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

// WebAppSettings represents complex application settings
type WebAppSettings struct {
	Theme              string   `json:"theme"`
	Language           string   `json:"language"`
	NotificationsEmail bool     `json:"notifications_email"`
	NotificationsPush  bool     `json:"notifications_push"`
	Features           []string `json:"features"`
}

// APICredentials represents sensitive API credentials
type APICredentials struct {
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	SecretKey    string `json:"secret_key"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// UserProfile represents public user profile information
type UserProfile struct {
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
}

func main() {
	fmt.Println("UserPrefs Web Application Example")
	fmt.Println("=================================")

	// Initialize storage based on environment
	var store userprefs.Storage
	var err error

	storageType := getEnvOrDefault("STORAGE_TYPE", "memory")
	switch storageType {
	case "sqlite":
		dbPath := getEnvOrDefault("SQLITE_PATH", "webapp_prefs.db")
		store, err = storage.NewSQLiteStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite storage: %v", err)
		}
		fmt.Printf("✓ Using SQLite storage: %s\n", dbPath)
	case "memory":
		store = storage.NewMemoryStorage()
		fmt.Println("✓ Using memory storage (development mode)")
	default:
		log.Fatalf("Unsupported storage type: %s", storageType)
	}
	defer store.Close()

	// Initialize cache based on environment
	var appCache userprefs.Cache
	cacheType := getEnvOrDefault("CACHE_TYPE", "memory")
	switch cacheType {
	case "memory":
		appCache = cache.NewMemoryCache()
		fmt.Println("✓ Using memory cache")
	case "none":
		appCache = nil
		fmt.Println("✓ No cache configured")
	default:
		log.Fatalf("Unsupported cache type: %s", cacheType)
	}
	if appCache != nil {
		defer appCache.Close()
	}

	// Initialize encryption if key is provided
	var encryptionAdapter userprefs.EncryptionManager
	if encKey := os.Getenv("USERPREFS_ENCRYPTION_KEY"); encKey != "" {
		encryptionAdapter, err = userprefs.NewEncryptionAdapter()
		if err != nil {
			log.Fatalf("Failed to initialize encryption: %v", err)
		}
		fmt.Println("✓ Encryption enabled")
	} else {
		fmt.Println("ℹ Encryption disabled (no USERPREFS_ENCRYPTION_KEY set)")
	}

	// Create preference manager
	opts := []userprefs.Option{
		userprefs.WithStorage(store),
	}
	if appCache != nil {
		opts = append(opts, userprefs.WithCache(appCache))
	}
	if encryptionAdapter != nil {
		opts = append(opts, userprefs.WithEncryption(encryptionAdapter))
	}

	mgr := userprefs.New(opts...)

	// Define application preferences
	err = defineWebAppPreferences(mgr)
	if err != nil {
		log.Fatalf("Failed to define preferences: %v", err)
	}

	ctx := context.Background()

	// Simulate web application scenarios
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SCENARIO 1: New User Registration")
	fmt.Println(strings.Repeat("=", 50))
	simulateNewUser(ctx, mgr, "user_001")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SCENARIO 2: Existing User Login & Settings Update")
	fmt.Println(strings.Repeat("=", 50))
	simulateExistingUser(ctx, mgr, "user_002")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SCENARIO 3: API Integration Settings")
	fmt.Println(strings.Repeat("=", 50))
	simulateAPIIntegration(ctx, mgr, "user_003")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SCENARIO 4: Bulk Settings Export/Import")
	fmt.Println(strings.Repeat("=", 50))
	simulateBulkOperations(ctx, mgr, "user_004")

	fmt.Println("\n✅ Web application example completed successfully!")
}

func defineWebAppPreferences(mgr *userprefs.Manager) error {
	// Check if encryption is available
	encryptionAvailable := os.Getenv("USERPREFS_ENCRYPTION_KEY") != ""

	preferences := []userprefs.PreferenceDefinition{
		// Public user profile settings
		{
			Key:      "user_profile",
			Type:     userprefs.JSONType,
			Category: "profile",
			DefaultValue: UserProfile{
				DisplayName: "",
				Avatar:      "",
				Bio:         "",
				Location:    "",
			},
			Encrypted: false,
		},
		// General application settings
		{
			Key:      "app_settings",
			Type:     userprefs.JSONType,
			Category: "settings",
			DefaultValue: WebAppSettings{
				Theme:              "light",
				Language:           "en",
				NotificationsEmail: true,
				NotificationsPush:  false,
				Features:           []string{},
			},
			Encrypted: false,
		},
		// Sensitive API credentials
		{
			Key:          "api_credentials_github",
			Type:         userprefs.JSONType,
			Category:     "integrations",
			DefaultValue: APICredentials{},
			Encrypted:    encryptionAvailable, // Only encrypt if encryption is available
		},
		{
			Key:          "api_credentials_slack",
			Type:         userprefs.JSONType,
			Category:     "integrations",
			DefaultValue: APICredentials{},
			Encrypted:    encryptionAvailable, // Only encrypt if encryption is available
		},
		// Simple preferences
		{
			Key:          "email_verified",
			Type:         userprefs.BoolType,
			Category:     "account",
			DefaultValue: false,
			Encrypted:    false,
		},
		{
			Key:           "subscription_tier",
			Type:          userprefs.StringType,
			Category:      "billing",
			DefaultValue:  "free",
			AllowedValues: []interface{}{"free", "pro", "enterprise"},
			Encrypted:     false,
		},
		{
			Key:          "api_rate_limit",
			Type:         userprefs.IntType,
			Category:     "billing",
			DefaultValue: 100,
			Encrypted:    false,
		},
		// Sensitive personal information
		{
			Key:      "personal_info",
			Type:     userprefs.JSONType,
			Category: "account",
			DefaultValue: map[string]interface{}{
				"full_name": "",
				"phone":     "",
				"address":   "",
			},
			Encrypted: encryptionAvailable, // Only encrypt if encryption is available
		},
	}

	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			return fmt.Errorf("failed to define preference %s: %w", pref.Key, err)
		}
	}

	fmt.Printf("✓ Defined %d preference types", len(preferences))
	if encryptionAvailable {
		fmt.Printf(" (with encryption)\n")
	} else {
		fmt.Printf(" (without encryption - set USERPREFS_ENCRYPTION_KEY to enable)\n")
	}
	return nil
}

func simulateNewUser(ctx context.Context, mgr *userprefs.Manager, userID string) {
	fmt.Printf("👤 New user registration: %s\n", userID)

	// Check initial settings (should be defaults)
	appSettings, err := mgr.Get(ctx, userID, "app_settings")
	if err != nil {
		log.Printf("Error getting app settings: %v", err)
		return
	}

	fmt.Printf("📋 Default app settings: %+v\n", appSettings.Value)

	// Set initial user profile
	profile := UserProfile{
		DisplayName: "New User",
		Avatar:      "https://example.com/avatar.jpg",
		Bio:         "Just joined the platform!",
		Location:    "San Francisco, CA",
	}

	err = mgr.Set(ctx, userID, "user_profile", profile)
	if err != nil {
		log.Printf("Error setting user profile: %v", err)
		return
	}
	fmt.Printf("✓ User profile created\n")

	// Mark email as verified
	err = mgr.Set(ctx, userID, "email_verified", true)
	if err != nil {
		log.Printf("Error setting email verified: %v", err)
		return
	}
	fmt.Printf("✓ Email marked as verified\n")

	// Get all account preferences
	accountPrefs, err := mgr.GetByCategory(ctx, userID, "account")
	if err != nil {
		log.Printf("Error getting account preferences: %v", err)
		return
	}

	fmt.Printf("📊 Account preferences summary:\n")
	for key, pref := range accountPrefs {
		fmt.Printf("  %s: %v\n", key, pref.Value)
	}
}

func simulateExistingUser(ctx context.Context, mgr *userprefs.Manager, userID string) {
	fmt.Printf("🔄 Existing user session: %s\n", userID)

	// Set up some existing preferences first
	existingSettings := WebAppSettings{
		Theme:              "dark",
		Language:           "en",
		NotificationsEmail: true,
		NotificationsPush:  true,
		Features:           []string{"advanced_editor", "beta_features"},
	}

	err := mgr.Set(ctx, userID, "app_settings", existingSettings)
	if err != nil {
		log.Printf("Error setting app settings: %v", err)
		return
	}

	// User updates their theme preference
	fmt.Println("🎨 User changing theme from dark to light...")

	currentSettings, err := mgr.Get(ctx, userID, "app_settings")
	if err != nil {
		log.Printf("Error getting current settings: %v", err)
		return
	}

	// Update just the theme
	var settings WebAppSettings
	data, _ := json.Marshal(currentSettings.Value)
	json.Unmarshal(data, &settings)
	settings.Theme = "light"

	err = mgr.Set(ctx, userID, "app_settings", settings)
	if err != nil {
		log.Printf("Error updating theme: %v", err)
		return
	}
	fmt.Printf("✓ Theme updated to: %s\n", settings.Theme)

	// User upgrades subscription
	fmt.Println("💎 User upgrading to pro subscription...")
	err = mgr.Set(ctx, userID, "subscription_tier", "pro")
	if err != nil {
		log.Printf("Error updating subscription: %v", err)
		return
	}

	err = mgr.Set(ctx, userID, "api_rate_limit", 1000)
	if err != nil {
		log.Printf("Error updating rate limit: %v", err)
		return
	}
	fmt.Printf("✓ Subscription upgraded to pro (rate limit: 1000)\n")

	// Show updated billing preferences
	billingPrefs, err := mgr.GetByCategory(ctx, userID, "billing")
	if err != nil {
		log.Printf("Error getting billing preferences: %v", err)
		return
	}

	fmt.Printf("💳 Updated billing preferences:\n")
	for key, pref := range billingPrefs {
		fmt.Printf("  %s: %v\n", key, pref.Value)
	}
}

func simulateAPIIntegration(ctx context.Context, mgr *userprefs.Manager, userID string) {
	fmt.Printf("🔗 API integration setup: %s\n", userID)

	// Add GitHub integration
	githubCreds := APICredentials{
		Provider:     "github",
		APIKey:       "ghp_1234567890abcdef",
		SecretKey:    "github_secret_key_xyz",
		RefreshToken: "github_refresh_token_abc",
	}

	err := mgr.Set(ctx, userID, "api_credentials_github", githubCreds)
	if err != nil {
		log.Printf("Error setting GitHub credentials: %v", err)
		return
	}
	fmt.Printf("✓ GitHub integration configured\n")

	// Add Slack integration
	slackCreds := APICredentials{
		Provider:  "slack",
		APIKey:    "xoxb-slack-bot-token",
		SecretKey: "slack_signing_secret",
	}

	err = mgr.Set(ctx, userID, "api_credentials_slack", slackCreds)
	if err != nil {
		log.Printf("Error setting Slack credentials: %v", err)
		return
	}
	fmt.Printf("✓ Slack integration configured\n")

	// Retrieve and verify integrations (credentials should be decrypted automatically)
	fmt.Println("🔍 Verifying stored integrations...")

	integrationPrefs, err := mgr.GetByCategory(ctx, userID, "integrations")
	if err != nil {
		log.Printf("Error getting integrations: %v", err)
		return
	}

	fmt.Printf("🔌 Active integrations:\n")
	for key, pref := range integrationPrefs {
		// Show partial info for security
		var creds APICredentials
		data, _ := json.Marshal(pref.Value)
		json.Unmarshal(data, &creds)

		maskedKey := maskSecret(creds.APIKey)
		fmt.Printf("  %s: %s (key: %s)\n", key, creds.Provider, maskedKey)
	}
}

func simulateBulkOperations(ctx context.Context, mgr *userprefs.Manager, userID string) {
	fmt.Printf("📦 Bulk operations demonstration: %s\n", userID)

	// Set up multiple preferences for the user
	preferences := map[string]interface{}{
		"user_profile": UserProfile{
			DisplayName: "Bulk User",
			Avatar:      "https://example.com/bulk-avatar.jpg",
			Bio:         "User created via bulk operations",
			Location:    "Remote",
		},
		"app_settings": WebAppSettings{
			Theme:              "dark",
			Language:           "es",
			NotificationsEmail: false,
			NotificationsPush:  true,
			Features:           []string{"bulk_import", "export_data"},
		},
		"subscription_tier": "enterprise",
		"api_rate_limit":    5000,
		"email_verified":    true,
	}

	fmt.Println("📥 Setting multiple preferences...")
	for key, value := range preferences {
		err := mgr.Set(ctx, userID, key, value)
		if err != nil {
			log.Printf("Error setting %s: %v", key, err)
			continue
		}
		fmt.Printf("  ✓ %s\n", key)
	}

	// Export all user preferences
	fmt.Println("📤 Exporting all user preferences...")
	allPrefs, err := mgr.GetAll(ctx, userID)
	if err != nil {
		log.Printf("Error getting all preferences: %v", err)
		return
	}

	fmt.Printf("📋 User %s has %d preferences:\n", userID, len(allPrefs))
	for key, pref := range allPrefs {
		// Show category and type info
		fmt.Printf("  %s [%s, %s]\n", key, pref.Category, pref.Type)
	}

	// Demonstrate settings summary by category
	fmt.Println("📊 Settings summary by category:")
	categories := map[string]int{}
	for _, pref := range allPrefs {
		categories[pref.Category]++
	}

	for category, count := range categories {
		fmt.Printf("  %s: %d preferences\n", category, count)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
