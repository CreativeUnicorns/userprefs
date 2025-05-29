// Package encryption provides AES-256 encryption/decryption capabilities for user preferences.
// It includes secure key validation and environment variable management.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// MinKeyLength is the minimum required length for the input key material (32 bytes).
	// The actual AES key will be derived from this using SHA-256.
	MinKeyLength = 32
	// AESKeyLength is the required length for AES-256 keys (32 bytes).
	AESKeyLength = 32
	// EnvKeyName is the environment variable name for the encryption key.
	EnvKeyName = "USERPREFS_ENCRYPTION_KEY"
)

var (
	// ErrInvalidKeyLength is returned when the encryption key doesn't meet minimum length requirements.
	ErrInvalidKeyLength = errors.New("encryption key must be at least 32 bytes")
	// ErrKeyNotFound is returned when the encryption key environment variable is not set.
	ErrKeyNotFound = errors.New("encryption key not found in environment variable " + EnvKeyName)
	// ErrEncryptionFailed is returned when encryption operation fails.
	ErrEncryptionFailed = errors.New("encryption operation failed")
	// ErrDecryptionFailed is returned when decryption operation fails.
	ErrDecryptionFailed = errors.New("decryption operation failed")
	// ErrInvalidCiphertext is returned when the ciphertext is malformed or too short.
	ErrInvalidCiphertext = errors.New("invalid ciphertext: too short or malformed")
)

// Manager handles AES-256-GCM encryption and decryption operations.
// It validates the encryption key during initialization for fast-fail scenarios.
type Manager struct {
	key []byte
}

// deriveKey derives a 32-byte AES key from input key material using SHA-256.
// This allows keys of any length >= MinKeyLength to be used safely.
func deriveKey(keyMaterial []byte) []byte {
	hash := sha256.Sum256(keyMaterial)
	return hash[:]
}

// NewManager creates a new encryption manager with the key from environment variable.
// It validates the key strength and length during initialization.
// Returns an error if the key is missing or doesn't meet security requirements.
func NewManager() (*Manager, error) {
	keyStr := os.Getenv(EnvKeyName)
	if keyStr == "" {
		return nil, ErrKeyNotFound
	}

	keyMaterial := []byte(keyStr)
	if len(keyMaterial) < MinKeyLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d", ErrInvalidKeyLength, len(keyMaterial), MinKeyLength)
	}

	// Derive a proper 32-byte AES key from the input material
	derivedKey := deriveKey(keyMaterial)
	return &Manager{key: derivedKey}, nil
}

// NewManagerWithKey creates a new encryption manager with a provided key.
// This is primarily used for testing. In production, use NewManager() with environment variables.
func NewManagerWithKey(keyMaterial []byte) (*Manager, error) {
	if len(keyMaterial) < MinKeyLength {
		return nil, fmt.Errorf("%w: got %d bytes, need at least %d", ErrInvalidKeyLength, len(keyMaterial), MinKeyLength)
	}

	// Derive a proper 32-byte AES key from the input material
	derivedKey := deriveKey(keyMaterial)
	return &Manager{key: derivedKey}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext.
// The returned string contains the nonce prepended to the encrypted data, all base64-encoded.
func (m *Manager) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create cipher: %v", ErrEncryptionFailed, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create GCM: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: failed to generate nonce: %v", ErrEncryptionFailed, err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64-encoded ciphertext using AES-256-GCM and returns plaintext.
// Expects the ciphertext to contain the nonce prepended to the encrypted data.
func (m *Manager) Decrypt(encodedCiphertext string) (string, error) {
	if encodedCiphertext == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encodedCiphertext)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64: %v", ErrDecryptionFailed, err)
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create cipher: %v", ErrDecryptionFailed, err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create GCM: %v", ErrDecryptionFailed, err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: failed to decrypt: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// ValidateKey validates that the encryption key meets security requirements.
// This can be called early in application startup for fast-fail validation.
func ValidateKey() error {
	keyStr := os.Getenv(EnvKeyName)
	if keyStr == "" {
		return ErrKeyNotFound
	}

	if len([]byte(keyStr)) < MinKeyLength {
		return fmt.Errorf("%w: got %d bytes, need at least %d", ErrInvalidKeyLength, len([]byte(keyStr)), MinKeyLength)
	}

	return nil
}
