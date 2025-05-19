// Package userprefs defines the core types used in the user preferences management system.
package userprefs

import (
	"time"
)

// Preference represents a single user preference setting as stored and retrieved by the system.
// It encapsulates the user's specific value for a defined preference, along with metadata.
// JSON tags are included for serialization, typically used by storage implementations.
type Preference struct {
	// UserID is the unique identifier for the user to whom this preference belongs.
	UserID string `json:"user_id"`
	// Key is the unique string identifier for this preference, matching a PreferenceDefinition.Key.
	Key string `json:"key"`
	// Value is the actual value set by the user for this preference.
	// Its type should conform to the Type specified in the corresponding PreferenceDefinition.
	Value interface{} `json:"value"`
	// DefaultValue holds the default value for this preference, as defined in its PreferenceDefinition.
	// This field is populated by the Manager when a preference is retrieved or set,
	// ensuring consistency. It's typically used by the application if the user hasn't set a specific Value.
	DefaultValue interface{} `json:"default_value,omitempty"`
	// Type is the string representation of the preference's data type (e.g., "string", "bool", "int").
	// It corresponds to the Type in the PreferenceDefinition.
	Type string `json:"type"`
	// Category is an optional grouping string for the preference, from its PreferenceDefinition.
	Category string `json:"category,omitempty"`
	// UpdatedAt records the time when this preference was last set or modified in storage.
	UpdatedAt time.Time `json:"updated_at"`
}

// PreferenceDefinition defines the schema, constraints, and default behavior for a particular preference key.
// Instances of PreferenceDefinition are registered with the Manager to make preferences known to the system.
// JSON tags are included for potential serialization, though definitions are typically configured in code.
type PreferenceDefinition struct {
	// Key is the unique string identifier for the preference type.
	// This key is used to get, set, and define the preference.
	Key string `json:"key"`
	// Type is the expected data type of the preference's value (e.g., "string", "bool", "int", "float", "json").
	// The Manager uses this for basic type validation when a preference is set.
	Type string `json:"type"`
	// DefaultValue is the value to be used if a user has not explicitly set this preference.
	// The type of DefaultValue must match the specified Type.
	DefaultValue interface{} `json:"default_value,omitempty"`
	// Category is an optional string used to group related preferences. This can be useful for UI organization
	// or for retrieving related sets of preferences (e.g., via Manager.GetByCategory).
	Category string `json:"category,omitempty"`
	// AllowedValues, if provided, restricts the preference's value to one of the items in this slice.
	// The Manager checks against these values during Set operations if Type validation passes.
	// The types of elements in AllowedValues must match the specified Type.
	AllowedValues []interface{} `json:"allowed_values,omitempty"`
	// ValidateFunc is an optional custom function that can be provided to perform complex validation
	// on a preference's value when it is being set. It is called after basic type checking and
	// AllowedValues checks (if any). The function receives the value to be validated and should
	// return nil if validation passes, or an error if it fails.
	// The `json:"-"` tag indicates this field is not serialized to JSON.
	ValidateFunc func(value interface{}) error `json:"-"`
}

// Config holds the internal configuration for a Manager instance.
// It is populated by applying functional Options (e.g., WithStorage, WithCache)
// when a new Manager is created with New().
// This struct is not intended to be instantiated or modified directly by users of the package.
type Config struct {
	// storage is the persistence layer implementation (e.g., PostgresStorage, SQLiteStorage, MemoryStorage).
	storage Storage
	// cache is the optional caching layer implementation.
	cache Cache
	// logger is the logging interface used by the Manager.
	logger Logger
	// definitions stores all registered PreferenceDefinition instances, keyed by their PreferenceDefinition.Key.
	definitions map[string]PreferenceDefinition
}

// Option defines the signature for a functional option that configures a Manager instance.
// Functions of this type are passed to New() to customize the Manager's behavior,
// such as setting its storage backend, cache, or logger.
// Each Option function takes a pointer to a Config struct and modifies it.
type Option func(*Config)

// WithStorage is a functional option that sets the Storage implementation for the Manager.
// The provided Storage (s) will be used for persisting and retrieving user preferences.
// This is a mandatory option for a functional Manager.
func WithStorage(s Storage) Option {
	return func(c *Config) {
		c.storage = s
	}
}

// WithCache is a functional option that sets the Cache implementation for the Manager.
// If a Cache is provided, the Manager will use it to cache frequently accessed preferences,
// potentially improving performance by reducing load on the Storage backend.
// This option is optional.
func WithCache(cache Cache) Option {
	return func(c *Config) {
		c.cache = cache
	}
}

// WithLogger is a functional option that sets the Logger implementation for the Manager.
// The Manager will use the provided Logger for logging informational messages, warnings, and errors.
// If not set, a default logger (writing to os.Stderr) may be used.
// This option is optional.
func WithLogger(l Logger) Option {
	return func(c *Config) {
		c.logger = l
	}
}
