// examples/validation/main.go
// This example demonstrates the ValidateFunc feature of the userprefs library.
// ValidateFunc allows you to add custom validation logic to preference definitions
// beyond basic type checking and AllowedValues constraints.
package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	// Initialize storage
	store, err := storage.NewSQLiteStorage("validation_example.db")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Create preference manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
	)

	// Define preferences with various validation scenarios
	preferences := []userprefs.PreferenceDefinition{
		// 1. Range validation for integers
		{
			Key:          "page_size",
			Type:         userprefs.IntType,
			DefaultValue: 20,
			Category:     "display",
			ValidateFunc: func(value interface{}) error {
				pageSize := value.(int) // Type is already validated by the manager
				if pageSize < 1 || pageSize > 100 {
					return fmt.Errorf("page size must be between 1 and 100, got %d", pageSize)
				}
				return nil
			},
		},

		// 2. Range validation for floats
		{
			Key:          "volume_level",
			Type:         userprefs.FloatType,
			DefaultValue: 0.5,
			Category:     "audio",
			ValidateFunc: func(value interface{}) error {
				volume := value.(float64) // Type is already validated by the manager
				if volume < 0.0 || volume > 1.0 {
					return fmt.Errorf("volume level must be between 0.0 and 1.0, got %.2f", volume)
				}
				return nil
			},
		},

		// 3. String format validation (email)
		{
			Key:          "notification_email",
			Type:         userprefs.StringType,
			DefaultValue: "",
			Category:     "notifications",
			ValidateFunc: func(value interface{}) error {
				email := value.(string)
				if email == "" {
					return nil // Empty email is allowed
				}

				// Use regex for proper email validation
				emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
				if !emailRegex.MatchString(email) {
					return fmt.Errorf("invalid email format: %s", email)
				}
				return nil
			},
		},

		// 4. String length validation
		{
			Key:          "username",
			Type:         userprefs.StringType,
			DefaultValue: "",
			Category:     "profile",
			ValidateFunc: func(value interface{}) error {
				username := value.(string)
				if len(username) < 3 {
					return fmt.Errorf("username must be at least 3 characters long")
				}
				if len(username) > 20 {
					return fmt.Errorf("username must be no more than 20 characters long")
				}

				// Only allow alphanumeric characters and underscores
				usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
				if !usernameRegex.MatchString(username) {
					return fmt.Errorf("username can only contain letters, numbers, and underscores")
				}
				return nil
			},
		},

		// 5. Custom business logic validation
		{
			Key:          "theme_colors",
			Type:         userprefs.JSONType,
			DefaultValue: map[string]interface{}{"primary": "#007bff", "secondary": "#6c757d"},
			Category:     "appearance",
			ValidateFunc: func(value interface{}) error {
				colors, ok := value.(map[string]interface{})
				if !ok {
					return fmt.Errorf("theme colors must be a JSON object")
				}

				// Ensure required color keys exist
				requiredColors := []string{"primary", "secondary"}
				for _, colorKey := range requiredColors {
					if _, exists := colors[colorKey]; !exists {
						return fmt.Errorf("missing required color: %s", colorKey)
					}
				}

				// Validate hex color format
				hexColorRegex := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
				for colorKey, colorValue := range colors {
					colorStr, ok := colorValue.(string)
					if !ok {
						return fmt.Errorf("color value for %s must be a string", colorKey)
					}
					if !hexColorRegex.MatchString(colorStr) {
						return fmt.Errorf("invalid hex color format for %s: %s", colorKey, colorStr)
					}
				}
				return nil
			},
		},

		// 6. Conditional validation based on other logic
		{
			Key:          "api_endpoint",
			Type:         userprefs.StringType,
			DefaultValue: "https://api.example.com",
			Category:     "integration",
			ValidateFunc: func(value interface{}) error {
				endpoint := value.(string)

				// Must be a valid URL
				if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
					return fmt.Errorf("API endpoint must start with http:// or https://")
				}

				// Must not be localhost in production (example business rule)
				if strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1") {
					return fmt.Errorf("localhost endpoints are not allowed in production")
				}

				return nil
			},
		},
	}

	// Register all preferences
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Fatalf("Failed to define preference %s: %v", pref.Key, err)
		}
	}

	ctx := context.Background()
	userID := "validation_demo_user"

	log.Println("=== UserPrefs Validation Demo ===\n")

	// Test each validation scenario
	testValidation(mgr, ctx, userID)
}

func testValidation(mgr *userprefs.Manager, ctx context.Context, userID string) {
	// Test 1: Page size validation
	log.Println("1. Testing page size validation (range: 1-100)")
	testSet(mgr, ctx, userID, "page_size", 25, true)   // Valid
	testSet(mgr, ctx, userID, "page_size", 0, false)   // Invalid: too low
	testSet(mgr, ctx, userID, "page_size", 150, false) // Invalid: too high

	// Test 2: Volume level validation
	log.Println("\n2. Testing volume level validation (range: 0.0-1.0)")
	testSet(mgr, ctx, userID, "volume_level", 0.75, true)  // Valid
	testSet(mgr, ctx, userID, "volume_level", -0.1, false) // Invalid: too low
	testSet(mgr, ctx, userID, "volume_level", 1.5, false)  // Invalid: too high

	// Test 3: Email validation
	log.Println("\n3. Testing email validation")
	testSet(mgr, ctx, userID, "notification_email", "user@example.com", true) // Valid
	testSet(mgr, ctx, userID, "notification_email", "", true)                 // Valid: empty allowed
	testSet(mgr, ctx, userID, "notification_email", "invalid-email", false)   // Invalid: bad format
	testSet(mgr, ctx, userID, "notification_email", "user@", false)           // Invalid: incomplete

	// Test 4: Username validation
	log.Println("\n4. Testing username validation")
	testSet(mgr, ctx, userID, "username", "john_doe123", true)                        // Valid
	testSet(mgr, ctx, userID, "username", "ab", false)                                // Invalid: too short
	testSet(mgr, ctx, userID, "username", "this_is_way_too_long_for_username", false) // Invalid: too long
	testSet(mgr, ctx, userID, "username", "user@name", false)                         // Invalid: contains @

	// Test 5: Theme colors validation
	log.Println("\n5. Testing theme colors validation")
	validColors := map[string]interface{}{
		"primary":   "#ff5733",
		"secondary": "#33ff57",
	}
	testSet(mgr, ctx, userID, "theme_colors", validColors, true) // Valid

	invalidColors1 := map[string]interface{}{
		"primary": "#ff5733", // Missing secondary
	}
	testSet(mgr, ctx, userID, "theme_colors", invalidColors1, false) // Invalid: missing required color

	invalidColors2 := map[string]interface{}{
		"primary":   "not-a-hex-color",
		"secondary": "#33ff57",
	}
	testSet(mgr, ctx, userID, "theme_colors", invalidColors2, false) // Invalid: bad hex format

	// Test 6: API endpoint validation
	log.Println("\n6. Testing API endpoint validation")
	testSet(mgr, ctx, userID, "api_endpoint", "https://api.production.com", true) // Valid
	testSet(mgr, ctx, userID, "api_endpoint", "http://api.staging.com", true)     // Valid
	testSet(mgr, ctx, userID, "api_endpoint", "ftp://api.com", false)             // Invalid: not http/https
	testSet(mgr, ctx, userID, "api_endpoint", "https://localhost:8080", false)    // Invalid: localhost not allowed

	log.Println("\n=== Validation Demo Complete ===")
}

func testSet(mgr *userprefs.Manager, ctx context.Context, userID, key string, value interface{}, expectSuccess bool) {
	err := mgr.Set(ctx, userID, key, value)

	if expectSuccess {
		if err != nil {
			log.Printf("  ✗ Expected success but got error for %s = %v: %v", key, value, err)
		} else {
			log.Printf("  ✓ Successfully set %s = %v", key, value)
		}
	} else {
		if err != nil {
			log.Printf("  ✓ Correctly rejected %s = %v: %v", key, value, err)
		} else {
			log.Printf("  ✗ Expected validation error but succeeded for %s = %v", key, value)
		}
	}
}
