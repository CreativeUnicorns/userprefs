package userprefs

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptionAdapter(t *testing.T) {
	key := []byte("this-is-a-32-byte-key-for-test!!")

	adapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)
	require.NotNil(t, adapter)

	plaintext := "sensitive data"
	encrypted, err := adapter.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := adapter.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptionAdapterWithEnv(t *testing.T) {
	key := "this-is-a-32-byte-key-for-test!!"
	os.Setenv("USERPREFS_ENCRYPTION_KEY", key)
	defer os.Unsetenv("USERPREFS_ENCRYPTION_KEY")

	adapter, err := NewEncryptionAdapter()
	require.NoError(t, err)
	require.NotNil(t, adapter)

	plaintext := "sensitive data"
	encrypted, err := adapter.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := adapter.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestManagerWithEncryption(t *testing.T) {
	// Create storage and encryption
	storage := NewMockStorage()
	key := []byte("this-is-a-32-byte-key-for-test!!")
	encryptionAdapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)

	// Create manager with encryption
	manager := New(
		WithStorage(storage),
		WithEncryption(encryptionAdapter),
	)

	// Define an encrypted preference
	err = manager.DefinePreference(PreferenceDefinition{
		Key:          "secret_token",
		Type:         StringType,
		DefaultValue: "",
		Encrypted:    true,
	})
	require.NoError(t, err)

	// Define a non-encrypted preference for comparison
	err = manager.DefinePreference(PreferenceDefinition{
		Key:          "public_setting",
		Type:         StringType,
		DefaultValue: "default",
		Encrypted:    false,
	})
	require.NoError(t, err)

	ctx := context.Background()
	userID := "user123"

	// Set encrypted preference
	secretValue := "super-secret-token-12345"
	t.Logf("Setting encrypted preference: %q", secretValue)

	// Check if definition is correct
	def, exists := manager.GetDefinition("secret_token")
	require.True(t, exists, "Definition should exist")
	t.Logf("Preference definition - Encrypted: %v, Type: %s", def.Encrypted, def.Type)

	err = manager.Set(ctx, userID, "secret_token", secretValue)
	require.NoError(t, err)

	// Set non-encrypted preference
	publicValue := "public-value"
	err = manager.Set(ctx, userID, "public_setting", publicValue)
	require.NoError(t, err)

	// Get encrypted preference - should return decrypted value
	pref, err := manager.Get(ctx, userID, "secret_token")
	require.NoError(t, err)
	assert.Equal(t, secretValue, pref.Value)

	// Get non-encrypted preference
	pref, err = manager.Get(ctx, userID, "public_setting")
	require.NoError(t, err)
	assert.Equal(t, publicValue, pref.Value)

	// Verify that the encrypted value is actually encrypted in storage
	storedPref, err := storage.Get(ctx, userID, "secret_token")
	require.NoError(t, err)
	t.Logf("Secret value: %q", secretValue)
	t.Logf("Stored value: %q (type: %T)", storedPref.Value, storedPref.Value)
	assert.NotEqual(t, secretValue, storedPref.Value) // Should be encrypted
	assert.IsType(t, "", storedPref.Value)            // Should be a string (base64 encoded)

	// Verify that the non-encrypted value is stored as-is
	storedPref, err = storage.Get(ctx, userID, "public_setting")
	require.NoError(t, err)
	assert.Equal(t, publicValue, storedPref.Value) // Should be the same
}

func TestEncryptedPreferenceTypes(t *testing.T) {
	storage := NewMockStorage()
	key := []byte("this-is-a-32-byte-key-for-test!!")
	encryptionAdapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)

	manager := New(
		WithStorage(storage),
		WithEncryption(encryptionAdapter),
	)

	// Define encrypted preferences of different types
	definitions := []PreferenceDefinition{
		{Key: "encrypted_string", Type: StringType, DefaultValue: "", Encrypted: true},
		{Key: "encrypted_bool", Type: BoolType, DefaultValue: false, Encrypted: true},
		{Key: "encrypted_int", Type: IntType, DefaultValue: 0, Encrypted: true},
		{Key: "encrypted_float", Type: FloatType, DefaultValue: 0.0, Encrypted: true},
		{Key: "encrypted_json", Type: JSONType, DefaultValue: map[string]interface{}{}, Encrypted: true},
	}

	for _, def := range definitions {
		err := manager.DefinePreference(def)
		require.NoError(t, err)
	}

	ctx := context.Background()
	userID := "user123"

	// Test different value types
	testCases := []struct {
		key      string
		value    interface{}
		expected interface{} // Expected value after encryption/decryption round trip
	}{
		{"encrypted_string", "secret string", "secret string"},
		{"encrypted_bool", true, true},
		{"encrypted_int", 42, 42},
		{"encrypted_float", 3.14159, 3.14159},
		{"encrypted_json", map[string]interface{}{"secret": "data", "number": 123}, map[string]interface{}{"secret": "data", "number": float64(123)}}, // JSON converts int to float64
	}

	// Set all values
	for _, tc := range testCases {
		err := manager.Set(ctx, userID, tc.key, tc.value)
		require.NoError(t, err, "Failed to set %s", tc.key)
	}

	// Get all values and verify they match
	for _, tc := range testCases {
		pref, err := manager.Get(ctx, userID, tc.key)
		require.NoError(t, err, "Failed to get %s", tc.key)
		expectedValue := tc.expected
		if expectedValue == nil {
			expectedValue = tc.value
		}
		assert.Equal(t, expectedValue, pref.Value, "Value mismatch for %s", tc.key)
	}
}

func TestEncryptionRequiredValidation(t *testing.T) {
	storage := NewMockStorage()

	// Create manager without encryption
	manager := New(WithStorage(storage))

	// Try to define an encrypted preference without encryption manager
	err := manager.DefinePreference(PreferenceDefinition{
		Key:       "secret",
		Type:      StringType,
		Encrypted: true,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEncryptionRequired)
}

func TestGetAllWithEncryption(t *testing.T) {
	storage := NewMockStorage()
	key := []byte("this-is-a-32-byte-key-for-test!!")
	encryptionAdapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)

	manager := New(
		WithStorage(storage),
		WithEncryption(encryptionAdapter),
	)

	// Define mixed encrypted and non-encrypted preferences
	definitions := []PreferenceDefinition{
		{Key: "secret1", Type: StringType, DefaultValue: "", Encrypted: true},
		{Key: "public1", Type: StringType, DefaultValue: "default", Encrypted: false},
		{Key: "secret2", Type: IntType, DefaultValue: 0, Encrypted: true},
	}

	for _, def := range definitions {
		err := manager.DefinePreference(def)
		require.NoError(t, err)
	}

	ctx := context.Background()
	userID := "user123"

	// Set some values
	err = manager.Set(ctx, userID, "secret1", "secret-value")
	require.NoError(t, err)
	err = manager.Set(ctx, userID, "public1", "public-value")
	require.NoError(t, err)
	err = manager.Set(ctx, userID, "secret2", 42)
	require.NoError(t, err)

	// Get all preferences
	prefs, err := manager.GetAll(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, prefs, 3)

	// Verify values are correctly decrypted
	assert.Equal(t, "secret-value", prefs["secret1"].Value)
	assert.Equal(t, "public-value", prefs["public1"].Value)
	assert.Equal(t, 42, prefs["secret2"].Value)
}

func TestGetByCategoryWithEncryption(t *testing.T) {
	storage := NewMockStorage()
	key := []byte("this-is-a-32-byte-key-for-test!!")
	encryptionAdapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)

	manager := New(
		WithStorage(storage),
		WithEncryption(encryptionAdapter),
	)

	// Define preferences in the same category
	definitions := []PreferenceDefinition{
		{Key: "secret_api_key", Type: StringType, Category: "api", DefaultValue: "", Encrypted: true},
		{Key: "public_endpoint", Type: StringType, Category: "api", DefaultValue: "", Encrypted: false},
	}

	for _, def := range definitions {
		err := manager.DefinePreference(def)
		require.NoError(t, err)
	}

	ctx := context.Background()
	userID := "user123"

	// Set values
	err = manager.Set(ctx, userID, "secret_api_key", "sk-1234567890")
	require.NoError(t, err)
	err = manager.Set(ctx, userID, "public_endpoint", "https://api.example.com")
	require.NoError(t, err)

	// Get by category
	prefs, err := manager.GetByCategory(ctx, userID, "api")
	require.NoError(t, err)
	assert.Len(t, prefs, 2)

	// Verify values are correctly decrypted
	assert.Equal(t, "sk-1234567890", prefs["secret_api_key"].Value)
	assert.Equal(t, "https://api.example.com", prefs["public_endpoint"].Value)
}

func TestCacheWithEncryption(t *testing.T) {
	storage := NewMockStorage()
	cache := NewMockCache()
	key := []byte("this-is-a-32-byte-key-for-test!!")
	encryptionAdapter, err := NewEncryptionAdapterWithKey(key)
	require.NoError(t, err)

	manager := New(
		WithStorage(storage),
		WithCache(cache),
		WithEncryption(encryptionAdapter),
	)

	// Define an encrypted preference
	err = manager.DefinePreference(PreferenceDefinition{
		Key:          "cached_secret",
		Type:         StringType,
		DefaultValue: "",
		Encrypted:    true,
	})
	require.NoError(t, err)

	ctx := context.Background()
	userID := "user123"
	secretValue := "cached-secret-value"

	// Set the preference (should cache the decrypted value)
	err = manager.Set(ctx, userID, "cached_secret", secretValue)
	require.NoError(t, err)

	// Get the preference (should hit cache and return decrypted value)
	pref, err := manager.Get(ctx, userID, "cached_secret")
	require.NoError(t, err)
	assert.Equal(t, secretValue, pref.Value)

	// Verify the storage still has the encrypted value
	storedPref, err := storage.Get(ctx, userID, "cached_secret")
	require.NoError(t, err)
	assert.NotEqual(t, secretValue, storedPref.Value) // Should be encrypted in storage
}
