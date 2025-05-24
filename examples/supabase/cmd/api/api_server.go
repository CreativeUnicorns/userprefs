// api_server.go - HTTP API server example for Supabase integration
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

type APIServer struct {
	manager userprefs.Manager
	port    string
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewAPIServer(manager userprefs.Manager, port string) *APIServer {
	return &APIServer{
		manager: manager,
		port:    port,
	}
}

func (s *APIServer) handleGetPreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	key := r.URL.Query().Get("key")

	if userID == "" || key == "" {
		s.respondError(w, "user_id and key parameters are required", http.StatusBadRequest)
		return
	}

	pref, err := s.manager.Get(context.Background(), userID, key)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to get preference: %v", err), http.StatusInternalServerError)
		return
	}

	s.respondSuccess(w, pref)
}

func (s *APIServer) handleSetPreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID string      `json:"user_id"`
		Key    string      `json:"key"`
		Value  interface{} `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" || req.Key == "" {
		s.respondError(w, "user_id and key are required", http.StatusBadRequest)
		return
	}

	err := s.manager.Set(context.Background(), req.UserID, req.Key, req.Value)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to set preference: %v", err), http.StatusInternalServerError)
		return
	}

	s.respondSuccess(w, map[string]string{"status": "updated"})
}

func (s *APIServer) handleGetAllPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		s.respondError(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	prefs, err := s.manager.GetAll(context.Background(), userID)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to get preferences: %v", err), http.StatusInternalServerError)
		return
	}

	s.respondSuccess(w, prefs)
}

func (s *APIServer) handleGetByCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	category := r.URL.Query().Get("category")

	if userID == "" || category == "" {
		s.respondError(w, "user_id and category parameters are required", http.StatusBadRequest)
		return
	}

	prefs, err := s.manager.GetByCategory(context.Background(), userID, category)
	if err != nil {
		s.respondError(w, fmt.Sprintf("Failed to get preferences by category: %v", err), http.StatusInternalServerError)
		return
	}

	s.respondSuccess(w, prefs)
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondSuccess(w, map[string]string{"status": "healthy", "service": "userprefs-supabase"})
}

func (s *APIServer) respondSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (s *APIServer) respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

func (s *APIServer) setupRoutes() {
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/preferences", s.handleGetPreference)
	http.HandleFunc("/preferences/set", s.handleSetPreference)
	http.HandleFunc("/preferences/all", s.handleGetAllPreferences)
	http.HandleFunc("/preferences/category", s.handleGetByCategory)

	// Serve a simple HTML page for testing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		html := `<!DOCTYPE html>
<html>
<head>
    <title>UserPrefs Supabase API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 4px; }
        code { background: #e0e0e0; padding: 2px 4px; border-radius: 2px; }
    </style>
</head>
<body>
    <h1>UserPrefs Supabase API Server</h1>
    <p>A REST API for managing user preferences with Supabase backend.</p>
    
    <h2>Available Endpoints:</h2>
    
    <div class="endpoint">
        <strong>GET /health</strong><br>
        Health check endpoint
    </div>
    
    <div class="endpoint">
        <strong>GET /preferences?user_id=USER&key=KEY</strong><br>
        Get a specific preference for a user
    </div>
    
    <div class="endpoint">
        <strong>POST /preferences/set</strong><br>
        Set a preference. Body: <code>{"user_id": "USER", "key": "KEY", "value": VALUE}</code>
    </div>
    
    <div class="endpoint">
        <strong>GET /preferences/all?user_id=USER</strong><br>
        Get all preferences for a user
    </div>
    
    <div class="endpoint">
        <strong>GET /preferences/category?user_id=USER&category=CATEGORY</strong><br>
        Get all preferences in a category for a user
    </div>
    
    <h2>Example Usage:</h2>
    <pre>
# Get user profile
curl "http://localhost:8080/preferences?user_id=demo_user&key=user_profile"

# Set theme preference
curl -X POST "http://localhost:8080/preferences/set" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "demo_user", "key": "theme", "value": "dark"}'

# Get all preferences
curl "http://localhost:8080/preferences/all?user_id=demo_user"
    </pre>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, html)
	})
}

func (s *APIServer) Start() error {
	s.setupRoutes()
	log.Printf("🚀 Starting UserPrefs API server on port %s", s.port)
	log.Printf("📖 Visit http://localhost:%s for API documentation", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Get configuration from environment
	supabaseURL := os.Getenv("SUPABASE_URL")
	dbURL := os.Getenv("SUPABASE_DB_URL")
	port := os.Getenv("APP_PORT")

	if port == "" {
		port = "8080"
	}

	if supabaseURL == "" || dbURL == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_DB_URL environment variables are required")
	}

	// Initialize storage
	store, err := storage.NewPostgresStorage(storage.WithPostgresDSN(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to Supabase database: %v", err)
	}
	defer store.Close()

	// Initialize cache
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
			log.Println("✅ Using Redis cache")
		}
	} else {
		cacheInstance = cache.NewMemoryCache()
		log.Println("✅ Using in-memory cache")
	}
	defer cacheInstance.Close()

	// Create preference manager
	mgr := userprefs.New(
		userprefs.WithStorage(store),
		userprefs.WithCache(cacheInstance),
	)

	// Define some basic preferences for the API
	preferences := []userprefs.PreferenceDefinition{
		{
			Key:           "theme",
			Type:          "enum",
			Category:      "appearance",
			DefaultValue:  "light",
			AllowedValues: []interface{}{"light", "dark", "auto"},
		},
		{
			Key:          "language",
			Type:         "string",
			Category:     "localization",
			DefaultValue: "en",
		},
		{
			Key:          "notifications_enabled",
			Type:         "boolean",
			Category:     "notifications",
			DefaultValue: true,
		},
		{
			Key:      "user_profile",
			Type:     "json",
			Category: "user",
			DefaultValue: map[string]interface{}{
				"name":     "",
				"email":    "",
				"timezone": "UTC",
			},
		},
	}

	// Register preferences
	log.Println("📋 Registering preference definitions...")
	for _, pref := range preferences {
		if err := mgr.DefinePreference(pref); err != nil {
			log.Printf("Warning: Failed to define preference %s: %v", pref.Key, err)
		}
	}

	// Create and start API server
	server := NewAPIServer(*mgr, port)
	log.Printf("🔗 Supabase URL: %s", supabaseURL)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
