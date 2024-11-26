// Package userprefs defines error variables used throughout the user preferences system.
package userprefs

import "errors"

// ErrInvalidInput indicates that the input parameters provided to a function are invalid.
var ErrInvalidInput = errors.New("invalid input parameters")

// ErrInvalidKey indicates that the provided preference key is invalid.
var ErrInvalidKey = errors.New("invalid preference key")

// ErrInvalidType indicates that the provided preference type is invalid.
var ErrInvalidType = errors.New("invalid preference type")

// ErrInvalidValue indicates that the provided preference value is invalid.
var ErrInvalidValue = errors.New("invalid preference value")

// ErrNotFound indicates that the requested preference was not found.
var ErrNotFound = errors.New("preference not found")

// ErrPreferenceNotDefined indicates that the preference has not been defined in the system.
var ErrPreferenceNotDefined = errors.New("preference not defined")

// ErrStorageUnavailable indicates that the storage backend is unavailable.
var ErrStorageUnavailable = errors.New("storage backend unavailable")

// ErrCacheUnavailable indicates that the cache backend is unavailable.
var ErrCacheUnavailable = errors.New("cache backend unavailable")
