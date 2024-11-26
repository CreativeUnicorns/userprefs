// validation.go
package userprefs

import (
	"encoding/json"
	"fmt"
)

var validTypes = map[string]bool{
	"string":  true,
	"boolean": true,
	"number":  true,
	"json":    true,
	"enum":    true,
}

func isValidType(t string) bool {
	return validTypes[t]
}

func validateValue(value interface{}, def PreferenceDefinition) error {
	switch def.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: expected string", ErrInvalidValue)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: expected boolean", ErrInvalidValue)
		}
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// Valid numeric types
		default:
			return fmt.Errorf("%w: expected number", ErrInvalidValue)
		}
	case "json":
		if _, err := json.Marshal(value); err != nil {
			return fmt.Errorf("%w: invalid JSON value", ErrInvalidValue)
		}
	case "enum":
		if len(def.AllowedValues) == 0 {
			return fmt.Errorf("%w: enum has no allowed values", ErrInvalidValue)
		}
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
	default:
		return fmt.Errorf("%w: unsupported type", ErrInvalidType)
	}
	return nil
}
