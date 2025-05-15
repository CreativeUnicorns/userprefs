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

## Storage Backends

### SQLite
```go
store, err := storage.NewSQLiteStorage("prefs.db")
```

### PostgreSQL
```go
store, err := storage.NewPostgresStorage("postgres://user:pass@localhost/dbname")
```

## Caching

### In-Memory
```go
cache := cache.NewMemoryCache()
mgr := userprefs.New(
    userprefs.WithStorage(store),
    userprefs.WithCache(cache),
)
```

### Redis
```go
cache, err := cache.NewRedisCache("localhost:6379", "", 0)
mgr := userprefs.New(
    userprefs.WithStorage(store),
    userprefs.WithCache(cache),
)
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