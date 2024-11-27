package userprefs

import (
	"testing"
)

func TestErrorVariables(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrInvalidInput", ErrInvalidInput, "invalid input parameters"},
		{"ErrInvalidKey", ErrInvalidKey, "invalid preference key"},
		{"ErrInvalidType", ErrInvalidType, "invalid preference type"},
		{"ErrInvalidValue", ErrInvalidValue, "invalid preference value"},
		{"ErrNotFound", ErrNotFound, "preference not found"},
		{"ErrPreferenceNotDefined", ErrPreferenceNotDefined, "preference not defined"},
		{"ErrStorageUnavailable", ErrStorageUnavailable, "storage backend unavailable"},
		{"ErrCacheUnavailable", ErrCacheUnavailable, "cache backend unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("Expected error message '%s', got '%s'", tt.expected, tt.err.Error())
			}
		})
	}
}
