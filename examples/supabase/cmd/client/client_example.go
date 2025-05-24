// client_example.go - Example client demonstrating API usage
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	baseURL = "http://localhost:8080"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PreferenceClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewPreferenceClient(baseURL string) *PreferenceClient {
	return &PreferenceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *PreferenceClient) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

func (c *PreferenceClient) GetPreference(userID, key string) (interface{}, error) {
	url := fmt.Sprintf("%s/preferences?user_id=%s&key=%s", c.baseURL, userID, key)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	return apiResp.Data, nil
}

func (c *PreferenceClient) SetPreference(userID, key string, value interface{}) error {
	payload := map[string]interface{}{
		"user_id": userID,
		"key":     key,
		"value":   value,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/preferences/set",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("API error: %s", apiResp.Error)
	}

	return nil
}

func (c *PreferenceClient) GetAllPreferences(userID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/preferences/all?user_id=%s", c.baseURL, userID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get all preferences: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	// Convert interface{} to map[string]interface{}
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("unexpected response format")
}

func (c *PreferenceClient) GetPreferencesByCategory(userID, category string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/preferences/category?user_id=%s&category=%s", c.baseURL, userID, category)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences by category: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	// Convert interface{} to map[string]interface{}
	if data, ok := apiResp.Data.(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("unexpected response format")
}

func main() {
	log.Println("🧪 UserPrefs Supabase API Client Example")
	log.Println("==========================================")

	client := NewPreferenceClient(baseURL)

	// Test health endpoint
	log.Println("🔍 Testing API health...")
	if err := client.Health(); err != nil {
		log.Fatalf("API health check failed: %v", err)
	}
	log.Println("✅ API is healthy")

	// Demo user
	userID := "client_demo_user"

	// Scenario 1: Set basic preferences
	log.Printf("\n📝 Setting preferences for user: %s", userID)

	preferences := map[string]interface{}{
		"theme":                 "dark",
		"language":              "en",
		"notifications_enabled": true,
		"user_profile": map[string]interface{}{
			"name":     "Demo User",
			"email":    "demo@example.com",
			"timezone": "America/New_York",
		},
	}

	for key, value := range preferences {
		if err := client.SetPreference(userID, key, value); err != nil {
			log.Printf("❌ Failed to set %s: %v", key, err)
		} else {
			log.Printf("✅ Set %s successfully", key)
		}
	}

	// Scenario 2: Get individual preferences
	log.Println("\n🔍 Retrieving individual preferences...")

	for key := range preferences {
		value, err := client.GetPreference(userID, key)
		if err != nil {
			log.Printf("❌ Failed to get %s: %v", key, err)
		} else {
			log.Printf("✅ %s: %v", key, formatValue(value))
		}
	}

	// Scenario 3: Get all preferences
	log.Println("\n📋 Retrieving all preferences...")

	allPrefs, err := client.GetAllPreferences(userID)
	if err != nil {
		log.Printf("❌ Failed to get all preferences: %v", err)
	} else {
		log.Printf("✅ Retrieved %d preferences:", len(allPrefs))
		for key, value := range allPrefs {
			log.Printf("  %s: %v", key, formatValue(value))
		}
	}

	// Scenario 4: Get preferences by category
	log.Println("\n🏷️  Retrieving preferences by category...")

	categories := []string{"appearance", "localization", "notifications", "user"}
	for _, category := range categories {
		prefs, err := client.GetPreferencesByCategory(userID, category)
		if err != nil {
			log.Printf("❌ Failed to get %s preferences: %v", category, err)
		} else {
			log.Printf("✅ %s category (%d preferences):", category, len(prefs))
			for key, value := range prefs {
				log.Printf("  %s: %v", key, formatValue(value))
			}
		}
	}

	// Scenario 5: Update preferences
	log.Println("\n🔄 Updating preferences...")

	updates := map[string]interface{}{
		"theme":                 "light",
		"language":              "es",
		"notifications_enabled": false,
	}

	for key, value := range updates {
		if err := client.SetPreference(userID, key, value); err != nil {
			log.Printf("❌ Failed to update %s: %v", key, err)
		} else {
			log.Printf("✅ Updated %s to %v", key, value)
		}
	}

	// Verify updates
	log.Println("\n✅ Verifying updates...")
	for key := range updates {
		value, err := client.GetPreference(userID, key)
		if err != nil {
			log.Printf("❌ Failed to verify %s: %v", key, err)
		} else {
			log.Printf("✅ %s is now: %v", key, formatValue(value))
		}
	}

	// Scenario 6: Performance test
	log.Println("\n⚡ Performance test...")

	start := time.Now()
	for i := 0; i < 50; i++ {
		_, err := client.GetPreference(userID, "theme")
		if err != nil {
			log.Printf("❌ Performance test failed at iteration %d: %v", i, err)
			break
		}
	}
	duration := time.Since(start)
	log.Printf("✅ 50 requests completed in %v (avg: %v per request)", duration, duration/50)

	// Scenario 7: Complex JSON preference
	log.Println("\n🏗️  Working with complex JSON preferences...")

	complexProfile := map[string]interface{}{
		"personal": map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       30,
			"interests": []string{"programming", "music", "travel"},
		},
		"settings": map[string]interface{}{
			"privacy": map[string]interface{}{
				"profileVisible": true,
				"emailVisible":   false,
				"shareData":      false,
			},
			"ui": map[string]interface{}{
				"fontSize":          16,
				"compactMode":       false,
				"animationsEnabled": true,
			},
		},
		"metadata": map[string]interface{}{
			"lastLogin":  time.Now().Format(time.RFC3339),
			"loginCount": 42,
			"isPremium":  true,
		},
	}

	if err := client.SetPreference(userID, "user_profile", complexProfile); err != nil {
		log.Printf("❌ Failed to set complex profile: %v", err)
	} else {
		log.Println("✅ Set complex user profile")

		// Retrieve and display
		profile, err := client.GetPreference(userID, "user_profile")
		if err != nil {
			log.Printf("❌ Failed to get complex profile: %v", err)
		} else {
			log.Println("✅ Retrieved complex profile:")
			if profileData, err := json.MarshalIndent(profile, "  ", "  "); err == nil {
				log.Printf("  %s", string(profileData))
			}
		}
	}

	log.Println("\n🎉 API client demo completed successfully!")
	log.Printf("💡 Tip: Start the API server with 'make run-api-local' to test this client")
}

func formatValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case bool:
		return fmt.Sprintf("%t", v)
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.2f", v)
	case map[string]interface{}:
		// For complex objects, show a summary
		count := len(v)
		if count == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d fields}", count)
	case []interface{}:
		return fmt.Sprintf("[%d items]", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}
