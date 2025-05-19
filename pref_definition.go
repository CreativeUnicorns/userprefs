package userprefs

// PreferenceType is a string alias for preference data types.
// Using a defined type allows for better type safety and discoverability
// compared to raw strings throughout the codebase.
// Deprecated: Use explicit string literals like "string", "bool", etc.
// The constants below are preferred for defining definition types.
type PreferenceType string

// Constants for standard preference types.
// These should be used when creating PreferenceDefinition instances to ensure consistency.
const (
	// StringType represents a preference value that is a string.
	StringType string = "string"
	// BoolType represents a preference value that is a boolean.
	BoolType string = "bool"
	// IntType represents a preference value that is an integer.
	IntType string = "int"
	// FloatType represents a preference value that is a floating-point number.
	FloatType string = "float"
	// JSONType represents a preference value that is a JSON object or array.
	// The actual Go type would typically be map[string]interface{} or []interface{}.
	JSONType string = "json"
)
