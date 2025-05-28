// Package userprefs provides an adapter for the encryption package.
package userprefs

import (
	"github.com/CreativeUnicorns/userprefs/encryption"
)

// EncryptionAdapter adapts the encryption.Manager to implement the EncryptionManager interface.
// This allows the encryption package to be used with the userprefs Manager.
type EncryptionAdapter struct {
	manager *encryption.Manager
}

// NewEncryptionAdapter creates a new EncryptionAdapter with encryption key from environment.
// It validates the encryption key during initialization for fast-fail scenarios.
// Returns an error if the key is missing or doesn't meet security requirements.
func NewEncryptionAdapter() (*EncryptionAdapter, error) {
	manager, err := encryption.NewManager()
	if err != nil {
		return nil, err
	}
	return &EncryptionAdapter{manager: manager}, nil
}

// NewEncryptionAdapterWithKey creates a new EncryptionAdapter with a provided key.
// This is primarily used for testing. In production, use NewEncryptionAdapter() with environment variables.
func NewEncryptionAdapterWithKey(key []byte) (*EncryptionAdapter, error) {
	manager, err := encryption.NewManagerWithKey(key)
	if err != nil {
		return nil, err
	}
	return &EncryptionAdapter{manager: manager}, nil
}

// Encrypt encrypts plaintext and returns the encrypted value as a string.
func (e *EncryptionAdapter) Encrypt(plaintext string) (string, error) {
	return e.manager.Encrypt(plaintext)
}

// Decrypt decrypts an encrypted value and returns the original plaintext.
func (e *EncryptionAdapter) Decrypt(encrypted string) (string, error) {
	return e.manager.Decrypt(encrypted)
}
