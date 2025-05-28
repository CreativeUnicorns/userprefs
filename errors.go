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

// ErrValidation indicates that a preference value failed a validation check.
var ErrValidation = errors.New("preference validation failed")

// ErrCacheClosed indicates that an operation was attempted on a cache that has been closed.
var ErrCacheClosed = errors.New("cache is closed")

// ErrPreferenceNotDefined indicates that the preference has not been defined in the system.
var ErrPreferenceNotDefined = errors.New("preference not defined")

// ErrStorageUnavailable indicates that the storage backend is unavailable.
var ErrStorageUnavailable = errors.New("storage backend unavailable")

// ErrCacheUnavailable indicates that the cache backend is unavailable.
var ErrCacheUnavailable = errors.New("cache backend unavailable")

// ErrAlreadyExists indicates that an attempt was made to create a resource that already exists.
var ErrAlreadyExists = errors.New("resource already exists")

// ErrSerialization indicates an error during data serialization or deserialization (e.g., JSON).
var ErrSerialization = errors.New("data serialization error")

// ErrInternal indicates an unexpected internal server error.
var ErrInternal = errors.New("internal server error")

// ErrEncryptionRequired indicates that an encryption manager is required but not configured.
var ErrEncryptionRequired = errors.New("encryption manager required for encrypted preferences")

// ErrEncryptionFailed indicates that an encryption or decryption operation failed.
var ErrEncryptionFailed = errors.New("encryption operation failed")
