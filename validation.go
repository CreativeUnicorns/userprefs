// Package userprefs provides validation functions for user preferences.
package userprefs

import (
	"encoding/json"
	"fmt"
)

// validTypes maps valid preference types to a boolean for quick lookup.
var validTypes = map[string]bool{
	StringType: true,
	BoolType:   true,
	IntType:    true,
	FloatType:  true,
	JSONType:   true,
}

// isValidType checks if the provided type is valid.
func isValidType(t string) bool {
	return validTypes[t]
}

// validateValue ensures that the value conforms to the preference definition.
func validateValue(value interface{}, def PreferenceDefinition) error {
	switch def.Type {
	case StringType:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: expected string", ErrInvalidValue)
		}
	case BoolType:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: expected boolean", ErrInvalidValue)
		}
	case IntType:
		switch value.(type) {
		case int, int32, int64:
			// Valid integer types
		default:
			return fmt.Errorf("%w: expected integer", ErrInvalidValue)
		}
	case FloatType:
		switch value.(type) {
		case float32, float64:
			// Valid float types
		default:
			return fmt.Errorf("%w: expected float", ErrInvalidValue)
		}
	case JSONType:
		if _, err := json.Marshal(value); err != nil {
			return fmt.Errorf("%w: invalid JSON value", ErrInvalidValue)
		}
	default:
		return fmt.Errorf("%w: unsupported type %s", ErrInvalidType, def.Type)
	}

	// Check allowed values if specified
	if len(def.AllowedValues) > 0 {
		found := false
		for _, allowed := range def.AllowedValues {
			if value == allowed {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: value not in allowed values", ErrInvalidValue)
		}
	}

	return nil
}
