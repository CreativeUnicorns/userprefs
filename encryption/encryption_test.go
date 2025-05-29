package encryption

import (
	"crypto/sha256"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// derivedKeyForTest computes the expected derived key for testing purposes
func derivedKeyForTest(keyMaterial []byte) []byte {
	hash := sha256.Sum256(keyMaterial)
	return hash[:]
}

func TestNewManager(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid key",
			envValue:    "this-is-a-32-byte-key-for-test!!",
			expectError: false,
		},
		{
			name:        "key too short",
			envValue:    "short",
			expectError: true,
			errorType:   ErrInvalidKeyLength,
		},
		{
			name:        "empty key",
			envValue:    "",
			expectError: true,
			errorType:   ErrKeyNotFound,
		},
		{
			name:        "exactly minimum length",
			envValue:    strings.Repeat("a", MinKeyLength),
			expectError: false,
		},
		{
			name:        "longer than minimum",
			envValue:    strings.Repeat("a", MinKeyLength+10),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			defer os.Unsetenv(EnvKeyName)

			// Set environment variable
			if tt.envValue != "" {
				os.Setenv(EnvKeyName, tt.envValue)
			} else {
				os.Unsetenv(EnvKeyName)
			}

			manager, err := NewManager()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				// Check that the key is the derived key, not the raw input
				expectedKey := derivedKeyForTest([]byte(tt.envValue))
				assert.Equal(t, expectedKey, manager.key)
			}
		})
	}
}

func TestNewManagerWithKey(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		expectError bool
		errorType   error
	}{
		{
			name:        "valid key",
			key:         []byte("this-is-a-32-byte-key-for-test!!"),
			expectError: false,
		},
		{
			name:        "key too short",
			key:         []byte("short"),
			expectError: true,
			errorType:   ErrInvalidKeyLength,
		},
		{
			name:        "exactly minimum length",
			key:         []byte(strings.Repeat("a", MinKeyLength)),
			expectError: false,
		},
		{
			name:        "nil key",
			key:         nil,
			expectError: true,
			errorType:   ErrInvalidKeyLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManagerWithKey(tt.key)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				// Check that the key is the derived key, not the raw input
				expectedKey := derivedKeyForTest(tt.key)
				assert.Equal(t, expectedKey, manager.key)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!!")
	manager, err := NewManagerWithKey(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "unicode text",
			plaintext: "Hello 世界! 🌍",
		},
		{
			name:      "json data",
			plaintext: `{"name":"John","age":30,"city":"New York"}`,
		},
		{
			name:      "long text",
			plaintext: strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 100),
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := manager.Encrypt(tt.plaintext)
			require.NoError(t, err)

			if tt.plaintext == "" {
				assert.Equal(t, "", encrypted)
				return
			}

			// Encrypted text should be different from plaintext
			assert.NotEqual(t, tt.plaintext, encrypted)
			// Encrypted text should be base64 encoded
			assert.True(t, len(encrypted) > 0)

			// Decrypt
			decrypted, err := manager.Decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptDecryptConsistency(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!!")
	manager, err := NewManagerWithKey(key)
	require.NoError(t, err)

	plaintext := "consistency test"

	// Encrypt the same plaintext multiple times
	encrypted1, err := manager.Encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := manager.Encrypt(plaintext)
	require.NoError(t, err)

	// Each encryption should produce different ciphertext (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2)

	// But both should decrypt to the same plaintext
	decrypted1, err := manager.Decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := manager.Decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDecryptInvalidData(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!!")
	manager, err := NewManagerWithKey(key)
	require.NoError(t, err)

	tests := []struct {
		name        string
		ciphertext  string
		expectError bool
		errorType   error
	}{
		{
			name:        "invalid base64",
			ciphertext:  "invalid-base64!@#",
			expectError: true,
			errorType:   ErrDecryptionFailed,
		},
		{
			name:        "too short ciphertext",
			ciphertext:  "dGVzdA==", // "test" in base64, but too short for nonce
			expectError: true,
			errorType:   ErrInvalidCiphertext,
		},
		{
			name:        "corrupted data",
			ciphertext:  "YWJjZGVmZ2hpams=", // valid base64 but invalid encrypted data
			expectError: true,
			errorType:   ErrInvalidCiphertext,
		},
		{
			name:        "empty string",
			ciphertext:  "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decrypted, err := manager.Decrypt(tt.ciphertext)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "", decrypted) // Empty input should return empty output
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid key",
			envValue:    "this-is-a-32-byte-key-for-test!!",
			expectError: false,
		},
		{
			name:        "key too short",
			envValue:    "short-key",
			expectError: true,
			errorType:   ErrInvalidKeyLength,
		},
		{
			name:        "empty key",
			envValue:    "",
			expectError: true,
			errorType:   ErrKeyNotFound,
		},
		{
			name:        "exactly minimum length",
			envValue:    strings.Repeat("a", MinKeyLength),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envValue != "" {
				os.Setenv(EnvKeyName, tt.envValue)
				defer os.Unsetenv(EnvKeyName)
			} else {
				os.Unsetenv(EnvKeyName)
			}

			err := ValidateKey()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorType != nil {
					assert.ErrorIs(t, err, tt.errorType)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCrossKeyCompatibility(t *testing.T) {
	key1 := []byte("this-is-a-32-byte-key-for-test!!")
	key2 := []byte("another-32-byte-key-for-testing!!")

	manager1, err := NewManagerWithKey(key1)
	require.NoError(t, err)

	manager2, err := NewManagerWithKey(key2)
	require.NoError(t, err)

	plaintext := "cross key test"

	// Encrypt with manager1
	encrypted, err := manager1.Encrypt(plaintext)
	require.NoError(t, err)

	// Try to decrypt with manager2 (should fail)
	_, err = manager2.Decrypt(encrypted)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func BenchmarkEncrypt(b *testing.B) {
	key := []byte("this-is-a-32-byte-key-for-test!!")
	manager, err := NewManagerWithKey(key)
	require.NoError(b, err)

	plaintext := "benchmark test data for encryption performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Encrypt(plaintext)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	key := []byte("this-is-a-32-byte-key-for-test!!")
	manager, err := NewManagerWithKey(key)
	require.NoError(b, err)

	plaintext := "benchmark test data for decryption performance"
	encrypted, err := manager.Encrypt(plaintext)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Decrypt(encrypted)
		if err != nil {
			b.Fatal(err)
		}
	}
}
