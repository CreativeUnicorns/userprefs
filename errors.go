// errors.go
package userprefs

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid input parameters")
	ErrInvalidKey           = errors.New("invalid preference key")
	ErrInvalidType          = errors.New("invalid preference type")
	ErrInvalidValue         = errors.New("invalid preference value")
	ErrNotFound             = errors.New("preference not found")
	ErrPreferenceNotDefined = errors.New("preference not defined")
	ErrStorageUnavailable   = errors.New("storage backend unavailable")
	ErrCacheUnavailable     = errors.New("cache backend unavailable")
)
