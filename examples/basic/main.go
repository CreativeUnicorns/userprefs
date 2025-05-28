// examples/basic/main.go
package main

import (
	"context"
	"fmt"
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
		// Example of custom validation function for page count
		{
			Key:          "page_count",
			Type:         userprefs.IntType,
			DefaultValue: 10,
			Category:     "display",
			ValidateFunc: func(value interface{}) error {
				// Type assertion - the manager already validates the type is int
				pageCount, ok := value.(int)
				if !ok {
					return fmt.Errorf("expected int, got %T", value)
				}

				// Custom range validation
				if pageCount < 1 || pageCount > 100 {
					return fmt.Errorf("page count must be between 1 and 100, got %d", pageCount)
				}

				return nil
			},
		},
		// Example of custom validation for email format
		{
			Key:          "notification_email",
			Type:         userprefs.StringType,
			DefaultValue: "",
			Category:     "notifications",
			ValidateFunc: func(value interface{}) error {
				email, ok := value.(string)
				if !ok {
					return fmt.Errorf("expected string, got %T", value)
				}

				// Simple email validation (in production, use a proper email validation library)
				if email != "" && !isValidEmail(email) {
					return fmt.Errorf("invalid email format: %s", email)
				}

				return nil
			},
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

	// Demonstrate validation success
	log.Println("\n--- Testing Validation ---")

	// Valid page count
	err = mgr.Set(ctx, userID, "page_count", 25)
	if err != nil {
		log.Printf("Failed to set valid page count: %v", err)
	} else {
		log.Println("✓ Successfully set page count to 25")
	}

	// Invalid page count (too high)
	err = mgr.Set(ctx, userID, "page_count", 150)
	if err != nil {
		log.Printf("✓ Validation correctly rejected page count 150: %v", err)
	} else {
		log.Println("✗ Validation should have rejected page count 150")
	}

	// Invalid page count (too low)
	err = mgr.Set(ctx, userID, "page_count", 0)
	if err != nil {
		log.Printf("✓ Validation correctly rejected page count 0: %v", err)
	} else {
		log.Println("✗ Validation should have rejected page count 0")
	}

	// Valid email
	err = mgr.Set(ctx, userID, "notification_email", "user@example.com")
	if err != nil {
		log.Printf("Failed to set valid email: %v", err)
	} else {
		log.Println("✓ Successfully set valid email")
	}

	// Invalid email
	err = mgr.Set(ctx, userID, "notification_email", "invalid-email")
	if err != nil {
		log.Printf("✓ Validation correctly rejected invalid email: %v", err)
	} else {
		log.Println("✗ Validation should have rejected invalid email")
	}

	// Get all preferences for category
	prefs, err := mgr.GetByCategory(ctx, userID, "media")
	if err != nil {
		log.Fatalf("Failed to get preferences: %v", err)
	}

	log.Printf("\nAll media preferences:")
	for key, pref := range prefs {
		log.Printf("  %s: %v", key, pref.Value)
	}
}

// Simple email validation function (for demonstration purposes)
func isValidEmail(email string) bool {
	// Very basic email validation - in production use a proper library
	if len(email) < 3 {
		return false
	}

	atIndex := -1
	dotIndex := -1

	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // Multiple @ symbols
			}
			atIndex = i
		} else if char == '.' && atIndex != -1 {
			dotIndex = i
		}
	}

	return atIndex > 0 && dotIndex > atIndex+1 && dotIndex < len(email)-1
}
