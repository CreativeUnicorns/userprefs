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

## Installation

```bash
go get github.com/CreativeUnicorns/userprefs
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/CreativeUnicorns/userprefs"
    "github.com/CreativeUnicorns/userprefs/storage"
)

func main() {
    // Initialize storage
    store, _ := storage.NewSQLiteStorage("prefs.db")
    defer store.Close()

    // Create manager
    mgr := userprefs.New(
        userprefs.WithStorage(store),
    )

    // Define preferences
    mgr.DefinePreference(userprefs.PreferenceDefinition{
        Key:          "theme",
        Type:         "enum",
        Category:     "appearance",
        DefaultValue: "dark",
        AllowedValues: []interface{}{
            "light", "dark", "system",
        },
    })

    // Set preference
    ctx := context.Background()
    mgr.Set(ctx, "user123", "theme", "light")

    // Get preference
    pref, _ := mgr.Get(ctx, "user123", "theme")
    fmt.Printf("Theme: %v\n", pref.Value)
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
4. Handle errors appropriately
5. Use context for timeout control
6. Close storage/cache connections on shutdown

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.