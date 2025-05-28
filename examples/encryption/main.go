// examples/encryption/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	fmt.Println("UserPrefs Encryption Example")
	fmt.Println("============================")

	// Initialize storage
	store := storage.NewMemoryStorage()
	defer store.Close()

	// Initialize cache for better performance
	memCache := cache.NewMemoryCache()
	defer memCache.Close()

	// Example 1: Create encryption adapter with a key
	fmt.Println("\n1. Creating encryption adapter with explicit key...")

	// For production, use a strong 32-byte key
	encryptionKey := []byte("this-is-a-32-byte-key-for-demo!!")
	encryptionAdapter, err := userprefs.NewEncryptionAdapterWithKey(encryptionKey)
	if err != nil {
		log.Fatalf("Failed to create encryption adapter: %v", err)
	}

	// Example 2: Create preference manager with encryption
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(memCache),
		userprefs.WithEncryption(encryptionAdapter),
	)

	// Example 3: Define both encrypted and non-encrypted preferences
	fmt.Println("\n2. Defining preferences (some encrypted, some not)...")

	preferences := []userprefs.PreferenceDefinition{
		{
			Key:          "api_key",
			Type:         userprefs.StringType,
			Category:     "security",
			DefaultValue: "",
			Encrypted:    true, // This will be encrypted at rest
		},
		{
			Key:          "database_password",
			Type:         userprefs.StringType,
			Category:     "security",
			DefaultValue: "",
			Encrypted:    true, // This will be encrypted at rest
		},
		{
			Key:          "user_tokens",
			Type:         userprefs.JSONType,
			Category:     "security",
			DefaultValue: map[string]interface{}{},
			Encrypted:    true, // Complex data can be encrypted too
		},
		{
			Key:           "theme",
			Type:          userprefs.StringType,
			Category:      "appearance",
			DefaultValue:  "dark",
			Encrypted:     false, // This is public data, no need to encrypt
			AllowedValues: []interface{}{"light", "dark", "system"},
		},
		{
			Key:          "notifications_enabled",
			Type:         userprefs.BoolType,
			Category:     "settings",
			DefaultValue: true,
			Encrypted:    false, // This is public data, no need to encrypt
		},
	}

	// Register all preferences
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	ctx := context.Background()
	userID := "user123"

	// Example 4: Set encrypted preferences
	fmt.Println("\n3. Setting encrypted preferences...")

	// Set sensitive data that will be encrypted
	err = mgr.Set(ctx, userID, "api_key", "sk-1234567890abcdef")
	if err != nil {
		log.Fatalf("Failed to set API key: %v", err)
	}
	fmt.Println("✓ API key set (encrypted)")

	err = mgr.Set(ctx, userID, "database_password", "super-secret-db-password-123!")
	if err != nil {
		log.Fatalf("Failed to set database password: %v", err)
	}
	fmt.Println("✓ Database password set (encrypted)")

	// Set complex encrypted data
	userTokens := map[string]interface{}{
		"access_token":  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"refresh_token": "def456789abcdef0123456789abcdef01234567",
		"expires_at":    1640995200,
	}
	err = mgr.Set(ctx, userID, "user_tokens", userTokens)
	if err != nil {
		log.Fatalf("Failed to set user tokens: %v", err)
	}
	fmt.Println("✓ User tokens set (encrypted)")

	// Set non-encrypted preferences
	err = mgr.Set(ctx, userID, "theme", "light")
	if err != nil {
		log.Fatalf("Failed to set theme: %v", err)
	}
	fmt.Println("✓ Theme set (not encrypted)")

	// Example 5: Retrieve and verify encryption/decryption
	fmt.Println("\n4. Retrieving preferences...")

	// Get encrypted data (automatically decrypted)
	apiKeyPref, err := mgr.Get(ctx, userID, "api_key")
	if err != nil {
		log.Fatalf("Failed to get API key: %v", err)
	}
	fmt.Printf("✓ API key retrieved: %s\n", apiKeyPref.Value)

	dbPasswordPref, err := mgr.Get(ctx, userID, "database_password")
	if err != nil {
		log.Fatalf("Failed to get database password: %v", err)
	}
	fmt.Printf("✓ Database password retrieved: %s\n", dbPasswordPref.Value)

	tokensPref, err := mgr.Get(ctx, userID, "user_tokens")
	if err != nil {
		log.Fatalf("Failed to get user tokens: %v", err)
	}
	fmt.Printf("✓ User tokens retrieved: %+v\n", tokensPref.Value)

	// Get non-encrypted data
	themePref, err := mgr.Get(ctx, userID, "theme")
	if err != nil {
		log.Fatalf("Failed to get theme: %v", err)
	}
	fmt.Printf("✓ Theme retrieved: %s\n", themePref.Value)

	// Example 6: Demonstrate storage vs manager view
	fmt.Println("\n5. Verifying encryption in storage...")

	// Check what's actually stored (should be encrypted for sensitive data)
	storedApiKey, err := store.Get(ctx, userID, "api_key")
	if err != nil {
		log.Fatalf("Failed to get stored API key: %v", err)
	}
	fmt.Printf("✓ Raw stored API key (encrypted): %s\n", storedApiKey.Value)

	storedTheme, err := store.Get(ctx, userID, "theme")
	if err != nil {
		log.Fatalf("Failed to get stored theme: %v", err)
	}
	fmt.Printf("✓ Raw stored theme (not encrypted): %s\n", storedTheme.Value)

	// Example 7: Get all preferences by category
	fmt.Println("\n6. Getting all security preferences...")

	securityPrefs, err := mgr.GetByCategory(ctx, userID, "security")
	if err != nil {
		log.Fatalf("Failed to get security preferences: %v", err)
	}

	fmt.Println("Security preferences (automatically decrypted):")
	for key, pref := range securityPrefs {
		// Only show partial values for security
		value := fmt.Sprintf("%v", pref.Value)
		if len(value) > 20 {
			value = value[:20] + "..."
		}
		fmt.Printf("  %s: %s\n", key, value)
	}

	// Example 8: Demonstrate environment variable configuration
	fmt.Println("\n7. Environment variable configuration example...")

	// Set encryption key via environment variable
	os.Setenv("USERPREFS_ENCRYPTION_KEY", "another-32-byte-key-for-env-demo")
	defer os.Unsetenv("USERPREFS_ENCRYPTION_KEY")

	// Create adapter from environment
	envEncryptionAdapter, err := userprefs.NewEncryptionAdapter()
	if err != nil {
		log.Fatalf("Failed to create encryption adapter from env: %v", err)
	}

	// Create a new manager with env-based encryption
	envMgr := userprefs.New(
		userprefs.WithStorage(storage.NewMemoryStorage()),
		userprefs.WithEncryption(envEncryptionAdapter),
	)

	// Define and use a simple encrypted preference
	err = envMgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:       "env_secret",
		Type:      userprefs.StringType,
		Encrypted: true,
	})
	if err != nil {
		log.Fatalf("Failed to define env preference: %v", err)
	}

	err = envMgr.Set(ctx, userID, "env_secret", "secret-from-env-key")
	if err != nil {
		log.Fatalf("Failed to set env secret: %v", err)
	}

	envSecretPref, err := envMgr.Get(ctx, userID, "env_secret")
	if err != nil {
		log.Fatalf("Failed to get env secret: %v", err)
	}
	fmt.Printf("✓ Secret with env-based encryption: %s\n", envSecretPref.Value)

	fmt.Println("\n✅ Encryption example completed successfully!")
	fmt.Println("\nKey takeaways:")
	fmt.Println("- Encrypted preferences are automatically encrypted/decrypted")
	fmt.Println("- Cache stores decrypted values for performance")
	fmt.Println("- Storage contains encrypted values for security")
	fmt.Println("- Both simple and complex data types can be encrypted")
	fmt.Println("- Environment variables can configure encryption keys")
}
