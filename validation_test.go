package userprefs

import (
	"errors"
	"testing"
)

func TestIsValidType(t *testing.T) {
	validTypesList := []string{"string", "boolean", "number", "json", "enum"}
	invalidTypesList := []string{"invalid", "list", "", "integer"}

	for _, tt := range validTypesList {
		if !isValidType(tt) {
			t.Errorf("Expected type '%s' to be valid", tt)
		}
	}

	for _, tt := range invalidTypesList {
		if isValidType(tt) {
			t.Errorf("Expected type '%s' to be invalid", tt)
		}
	}
}

func TestValidateValue_String(t *testing.T) {
	def := PreferenceDefinition{
		Key:  "language",
		Type: "string",
	}

	err := validateValue("en", def)
	if err != nil {
		t.Errorf("Expected valid string, got error: %v", err)
	}

	err = validateValue(123, def)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue, got: %v", err)
	}
}

func TestValidateValue_Boolean(t *testing.T) {
	def := PreferenceDefinition{
		Key:  "notifications",
		Type: "boolean",
	}

	err := validateValue(true, def)
	if err != nil {
		t.Errorf("Expected valid boolean, got error: %v", err)
	}

	err = validateValue("yes", def)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue, got: %v", err)
	}
}

func TestValidateValue_Number(t *testing.T) {
	def := PreferenceDefinition{
		Key:  "volume",
		Type: "number",
	}

	err := validateValue(10, def)
	if err != nil {
		t.Errorf("Expected valid number, got error: %v", err)
	}

	err = validateValue(3.14, def)
	if err != nil {
		t.Errorf("Expected valid number, got error: %v", err)
	}

	err = validateValue("high", def)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue, got: %v", err)
	}
}

func TestValidateValue_JSON(t *testing.T) {
	def := PreferenceDefinition{
		Key:  "settings",
		Type: "json",
	}

	validJSON := map[string]interface{}{
		"key1": "value1",
		"key2": 2,
	}

	err := validateValue(validJSON, def)
	if err != nil {
		t.Errorf("Expected valid JSON, got error: %v", err)
	}

	invalidJSON := make(chan int) // Channels cannot be marshaled to JSON
	err = validateValue(invalidJSON, def)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue for invalid JSON, got: %v", err)
	}
}

func TestValidateValue_Enum(t *testing.T) {
	def := PreferenceDefinition{
		Key:           "theme",
		Type:          "enum",
		AllowedValues: []interface{}{"light", "dark", "system"},
	}

	err := validateValue("dark", def)
	if err != nil {
		t.Errorf("Expected valid enum value, got error: %v", err)
	}

	err = validateValue("blue", def)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue for invalid enum, got: %v", err)
	}

	// Enum with no allowed values
	defNoAllowed := PreferenceDefinition{
		Key:  "invalid_enum",
		Type: "enum",
	}

	err = validateValue("any", defNoAllowed)
	if !errors.Is(err, ErrInvalidValue) {
		t.Errorf("Expected ErrInvalidValue for enum with no allowed values, got: %v", err)
	}
}

func TestValidateValue_UnsupportedType(t *testing.T) {
	def := PreferenceDefinition{
		Key:  "unsupported",
		Type: "unsupported",
	}

	err := validateValue("value", def)
	if !errors.Is(err, ErrInvalidType) {
		t.Errorf("Expected ErrInvalidType, got: %v", err)
	}
}
