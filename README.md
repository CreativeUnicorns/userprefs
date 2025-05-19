# UserPrefs

`userprefs` is a flexible, concurrent-safe user preferences management system for Go applications. While originally designed for Discord bots, it can be used in any application requiring user-specific preference storage.

[![Go Reference](https://pkg.go.dev/badge/github.com/CreativeUnicorns/userprefs.svg)](https://pkg.go.dev/github.com/CreativeUnicorns/userprefs)
[![Go Report Card](https://goreportcard.com/badge/github.com/CreativeUnicorns/userprefs)](https://goreportcard.com/report/github.com/CreativeUnicorns/userprefs)

## Features

- Thread-safe preference management
- Multiple storage backends (PostgreSQL, SQLite)
- Optional caching (Redis, in-memory)
- Flexible type system (string, boolean, number, JSON, enum)
- Category-based organization
- Default values support
- Context-aware operations
- Discord bot framework agnostic
- Built and tested with Go 1.24
- Integrated structured logging using Go's standard `slog` for enhanced observability.

## Installation

```bash
go get github.com/CreativeUnicorns/userprefs
```

## Quick Start

```go
package main

import (
    "context"
    "errors" // Added for errors.Is
    "fmt"    // Added for Printf
    "log"    // Added for logging examples
    "github.com/CreativeUnicorns/userprefs"
    "github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
    // Initialize storage
    // Initialize storage. The database path is a direct argument.
    // Additional configurations like WAL mode or busy timeout can be set via functional options.
    store, err := storage.NewSQLiteStorage("prefs.db")
    if err != nil {
        log.Fatalf("Failed to initialize SQLite storage: %v", err)
    }
    defer store.Close()

    // Create manager
    // You can also pass a custom logger implementing userprefs.Logger, 
    // by default it uses a standard slog.Logger.
    mgr := userprefs.New(
        userprefs.WithStorage(store),
        // userprefs.WithLogger(yourCustomSlogCompatibleLogger),
    )

    // Define preferences
    err = mgr.DefinePreference(userprefs.PreferenceDefinition{
        Key:          "theme",
        Type:         "enum",
        Category:     "appearance",
        DefaultValue: "dark",
        AllowedValues: []interface{}{
            "light", "dark", "system",
        },
    })
    if err != nil {
        log.Printf("Warning: Failed to define preference 'theme': %v", err) // Or handle more gracefully
    }

    // Set preference
    ctx := context.Background()
    err = mgr.Set(ctx, "user123", "theme", "light")
    if err != nil {
        log.Printf("Failed to set preference 'theme' for user 'user123': %v", err)
        // Handle error, e.g., retry or inform user
    }

    // Get preference
    pref, err := mgr.Get(ctx, "user123", "theme")
    if err != nil {
        if errors.Is(err, userprefs.ErrNotFound) {
            log.Printf("Preference 'theme' not found for user 'user123'. Using default: %v", pref.DefaultValue)
            // The 'pref' variable will contain the default value if one was defined
        } else {
            log.Printf("Failed to get preference 'theme' for user 'user123': %v", err)
            // Handle other errors
            return
        }
    }
    fmt.Printf("Theme: %v\n", pref.Value)

    // Example of getting all preferences for a user (error handling omitted for brevity here, but should be done)
    // allPrefs, _ := mgr.GetAll(ctx, "user123")
    // for key, pVal := range allPrefs {
    // 	fmt.Printf("User123 - Key: %s, Value: %v\n", key, pVal.Value)
    // }
}
```

## Storage and Cache Backends

Below are examples demonstrating how to initialize and use different storage and cache backends with the `userprefs.Manager`.

### Storage Backends

#### SQLite Storage

SQLite is a good option for file-based persistence, suitable for smaller applications or local development.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	ctx := context.Background()

	// Initialize SQLite storage with the default settings
	sqliteStore, err := storage.NewSQLiteStorage("my_app_prefs.db")
	if err != nil {
		log.Fatalf("Failed to initialize SQLite storage: %v", err)
	}
	defer sqliteStore.Close() // Important to close the database connection

	// Create a manager with SQLite storage
	mgr := userprefs.New(userprefs.WithStorage(sqliteStore))

	// Define a preference
	err = mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "notify_via_email",
		Type:         userprefs.PreferenceTypeBoolean,
		DefaultValue: true,
		Category:     "notifications",
	})
	if err != nil {
		log.Printf("Failed to define preference: %v", err)
	}

	// Set a preference for a user
	userID := "user789"
	err = mgr.Set(ctx, userID, "notify_via_email", false)
	if err != nil {
		log.Printf("Failed to set preference: %v", err)
	}

	// Get the preference
	pref, err := mgr.Get(ctx, userID, "notify_via_email")
	if err != nil {
		log.Printf("Failed to get preference: %v", err)
	} else {
		fmt.Printf("User %s: Notify via email: %v (Default: %v)\n", userID, pref.Value, pref.DefaultValue)
	}
}
```

#### PostgreSQL Storage

PostgreSQL is a robust relational database suitable for production environments requiring scalability and advanced features.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time" // Added time package

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	ctx := context.Background()

	// Initialize PostgreSQL storage using functional options
	// Replace with your actual PostgreSQL connection string
	dsn := "postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable"
	pgStore, err := storage.NewPostgresStorage(
		storage.WithPostgresDSN(dsn),
		storage.WithPostgresConnectTimeout(5*time.Second), // Example: Set a 5-second connection timeout
		// storage.WithPostgresMaxOpenConns(10), // Example: Set max open connections
		// storage.WithPostgresMaxIdleConns(5),   // Example: Set max idle connections
	)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL storage: %v", err)
	}
	defer pgStore.Close() // Important to close the database connection

	// Create a manager with PostgreSQL storage
	mgr := userprefs.New(userprefs.WithStorage(pgStore))

	// Define a preference
	err = mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "items_per_page",
		Type:         userprefs.PreferenceTypeNumber,
		DefaultValue: 25,
		Category:     "display",
	})
	if err != nil {
		log.Printf("Failed to define preference: %v", err)
	}

	// Set a preference for a user
	userID := "user101"
	err = mgr.Set(ctx, userID, "items_per_page", 50)
	if err != nil {
		log.Printf("Failed to set preference: %v", err)
	}

	// Get the preference
	pref, err := mgr.Get(ctx, userID, "items_per_page")
	if err != nil {
		log.Printf("Failed to get preference: %v", err)
	} else {
		fmt.Printf("User %s: Items per page: %v (Default: %v)\n", userID, pref.Value, pref.DefaultValue)
	}
}
```

#### In-Memory Storage

The in-memory storage is useful for testing, examples, or scenarios where persistence across application restarts is not required.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	ctx := context.Background()

	// Initialize In-Memory storage
	memStore := storage.NewMemoryStorage() // No error to handle for NewMemoryStorage
	// No defer memStore.Close() needed as it doesn't hold external resources

	// Create a manager with In-Memory storage
	mgr := userprefs.New(userprefs.WithStorage(memStore))

	// Define a preference
	err := mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "ui_language",
		Type:         userprefs.PreferenceTypeString,
		DefaultValue: "en-US",
		Category:     "regional",
	})
	if err != nil {
		log.Printf("Failed to define preference: %v", err)
	}

	// Set a preference for a user
	userID := "user456"
	err = mgr.Set(ctx, userID, "ui_language", "fr-CA")
	if err != nil {
		log.Printf("Failed to set preference: %v", err)
	}

	// Get the preference
	pref, err := mgr.Get(ctx, userID, "ui_language")
	if err != nil {
		log.Printf("Failed to get preference: %v", err)
	} else {
		log.Printf("User '%s' - Language: %v (Default: %v)\n", userID, pref.Value, pref.DefaultValue)
	}

	// Example: Initialize SQLite storage with custom options
	customSqliteStore, err := storage.NewSQLiteStorage(
		"my_app_prefs_custom.db",
		storage.WithSQLiteWAL(false), // Disable Write-Ahead Logging
		storage.WithSQLiteBusyTimeout(10*time.Second), // Set a 10-second busy timeout
		// storage.WithSQLiteJournalMode("DELETE"), // Example: Explicitly set journal mode
		// storage.WithSQLiteExtraParam("_cache_size", "-2000"), // Example: Set cache size (2MB)
	)
	if err != nil {
		log.Fatalf("Failed to initialize custom SQLite storage: %v", err)
	}
	defer customSqliteStore.Close()
	// Use customSqliteStore with userprefs.New...
}
```

### Cache Backends

Caching can significantly improve performance by reducing load on the primary storage backend.

#### In-Memory Cache

An in-memory cache is simple to set up and useful for single-instance applications.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	ctx := context.Background()

	// Initialize a primary storage (e.g., In-Memory for this example)
	primaryStore := storage.NewMemoryStorage()

	// Initialize In-Memory cache
	memCache := cache.NewMemoryCache(cache.WithDefaultExpiration(5 * time.Minute), cache.WithCleanupInterval(10 * time.Minute))
	// No defer memCache.Close() needed unless you want to stop its GC explicitly for some reason (usually not required).

	// Create a manager with storage and In-Memory cache
	mgr := userprefs.New(
		userprefs.WithStorage(primaryStore),
		userprefs.WithCache(memCache),
	)

	// Define a preference
	err := mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "auto_save_interval",
		Type:         userprefs.PreferenceTypeNumber,
		DefaultValue: 60, // seconds
		Category:     "editor",
	})
	if err != nil {
		log.Printf("Failed to define preference: %v", err)
	}

	// Set a preference for a user
	userID := "user777"
	err = mgr.Set(ctx, userID, "auto_save_interval", 120)
	if err != nil {
		log.Printf("Failed to set preference: %v", err)
	}

	// Get the preference (first time might hit storage, subsequent times should hit cache)
	pref, err := mgr.Get(ctx, userID, "auto_save_interval")
	if err != nil {
		log.Printf("Failed to get preference: %v", err)
	} else {
		fmt.Printf("User %s: Auto Save Interval: %v seconds (Default: %v)\n", userID, pref.Value, pref.DefaultValue)
	}

	// Get it again to demonstrate caching (behavior depends on cache TTL and manager logic)
	prefCached, _ := mgr.Get(ctx, userID, "auto_save_interval")
	fmt.Printf("User %s: Auto Save Interval (cached attempt): %v seconds\n", userID, prefCached.Value)
}
```

#### Redis Cache

Redis provides a fast, distributed cache suitable for multi-instance applications.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/CreativeUnicorns/userprefs"
	"github.com/CreativeUnicorns/userprefs/cache"
	"github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
	ctx := context.Background()

	// Initialize a primary storage (e.g., In-Memory for this example)
	primaryStore := storage.NewMemoryStorage()

	// Initialize Redis cache
	// Replace with your Redis server details
	redisAddr := "localhost:6379"
	redisPassword := "" // No password
	redisDB := 0       // Default DB

	// Initialize Redis cache using functional options
	redisCache, err := cache.NewRedisCache(
		cache.WithRedisAddress(redisAddr),
		cache.WithRedisPassword(redisPassword),
		cache.WithRedisDB(redisDB),
		cache.WithRedisDefaultTTL(5*time.Minute), // Example: Set a default TTL for cached items
		// cache.WithRedisMaxRetries(3),          // Example: Set max retries on connection failure
	)
	if err != nil {
		log.Fatalf("Failed to initialize Redis cache: %v", err)
	}
	defer redisCache.Close() // Important to close the Redis client connection

	// Create a manager with storage and Redis cache
	mgr := userprefs.New(
		userprefs.WithStorage(primaryStore),
		userprefs.WithCache(redisCache),
	)

	// Define a preference
	err = mgr.DefinePreference(userprefs.PreferenceDefinition{
		Key:          "session_timeout",
		Type:         userprefs.PreferenceTypeNumber,
		DefaultValue: 1800, // seconds (30 minutes)
		Category:     "security",
	})
	if err != nil {
		log.Printf("Failed to define preference: %v", err)
	}

	// Set a preference for a user
	userID := "user888"
	err = mgr.Set(ctx, userID, "session_timeout", 3600) // 1 hour
	if err != nil {
		log.Printf("Failed to set preference: %v", err)
	}

	// Get the preference
	pref, err := mgr.Get(ctx, userID, "session_timeout")
	if err != nil {
		log.Printf("Failed to get preference: %v", err)
	} else {
		fmt.Printf("User %s: Session Timeout: %v seconds (Default: %v)\n", userID, pref.Value, pref.DefaultValue)
	}
}
```

## Preference Types

- `string`: String values
- `boolean`: True/false values
- `number`: Numeric values
- `json`: Complex JSON structures
- `enum`: Predefined set of values

```go
// String preference
mgr.DefinePreference(userprefs.PreferenceDefinition{
    Key:          "nickname",
    Type:         "string",
    Category:     "profile",
    DefaultValue: "User",
})

// JSON preference
type Settings struct {
    Language string   `json:"language"`
    Features []string `json:"features"`
}

mgr.DefinePreference(userprefs.PreferenceDefinition{
    Key:      "settings",
    Type:     "json",
    Category: "system",
    DefaultValue: Settings{
        Language: "en",
        Features: []string{"basic"},
    },
})
```

## Discord Integration

The module is framework-agnostic and works with any Discord bot library. Add the `/preferences` command to your bot:

```go
// Get preference before command execution
prefs, _ := prefManager.Get(ctx, userID, "format")
format := prefs.Value.(string)

// Set preference from slash command
prefManager.Set(ctx, userID, "format", "mp4")

// Get all preferences by category
prefs, _ := prefManager.GetByCategory(ctx, userID, "media")
```

See the [examples](./examples) directory for complete Discord bot implementations.

## Best Practices

1. Define preferences at startup
2. Use categories to organize preferences
3. Always provide default values
4. Handle errors appropriately, checking for specific errors like `userprefs.ErrNotFound` and `userprefs.ErrSerialization` to build robust applications.
5. Use context for timeout control
6. Close storage/cache connections on shutdown

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.