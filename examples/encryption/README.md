# UserPrefs Encryption Example

This example demonstrates how to use the encryption features of the UserPrefs library to securely store sensitive user preferences.

## Overview

The encryption feature allows you to mark specific preferences as encrypted. When enabled:

- **Storage**: Contains encrypted values using AES-256-GCM encryption
- **Cache**: Contains decrypted values for performance (if caching is enabled)
- **Manager**: Automatically encrypts/decrypts values transparently

## Features Demonstrated

1. **Creating encryption adapters** with explicit keys and environment variables
2. **Defining encrypted preferences** alongside non-encrypted ones
3. **Automatic encryption/decryption** when setting and getting values
4. **Complex data type encryption** including JSON objects
5. **Storage vs Manager views** showing raw encrypted vs decrypted data
6. **Environment variable configuration** for production deployments

## Running the Example

```bash
# From the encryption example directory
go run main.go
```

## Key Security Concepts

### Encryption Key Requirements

- **Minimum length**: 32 bytes for AES-256 encryption
- **Environment variable**: `USERPREFS_ENCRYPTION_KEY`
- **Key rotation**: Requires application restart and data re-encryption

### What Gets Encrypted

Only preferences marked with `Encrypted: true` in their definition:

```go
{
    Key:       "api_key",
    Type:      userprefs.StringType,
    Encrypted: true,  // This preference will be encrypted
}
```

### Performance Considerations

- **Cache optimization**: Decrypted values are cached for performance
- **Encryption overhead**: Small latency cost for encrypt/decrypt operations
- **Memory usage**: Encrypted data takes slightly more storage space

## Production Usage

### Environment Setup

```bash
export USERPREFS_ENCRYPTION_KEY='your-32-byte-or-longer-encryption-key-here'
```

### Code Example

```go
// Create encryption adapter from environment
encryptionAdapter, err := userprefs.NewEncryptionAdapter()
if err != nil {
    log.Fatalf("Failed to create encryption adapter: %v", err)
}

// Create manager with encryption
mgr := userprefs.New(
    userprefs.WithStorage(yourStorage),
    userprefs.WithCache(yourCache),
    userprefs.WithEncryption(encryptionAdapter),
)

// Define encrypted preference
err = mgr.DefinePreference(userprefs.PreferenceDefinition{
    Key:       "api_secret",
    Type:      userprefs.StringType,
    Encrypted: true,
})
```

## Security Best Practices

1. **Strong keys**: Use cryptographically secure random keys of at least 32 bytes
2. **Environment variables**: Never hardcode encryption keys in source code
3. **Key rotation**: Plan for periodic key rotation in production
4. **Access control**: Limit access to encryption keys and encrypted storage
5. **Audit logging**: Monitor access to encrypted preferences

## What This Example Shows

The example output will demonstrate:

- ✅ Successful encryption/decryption of various data types
- ✅ Raw encrypted data in storage vs decrypted data from manager
- ✅ Environment variable configuration
- ✅ Mixed encrypted and non-encrypted preferences
- ✅ Category-based retrieval with automatic decryption

This ensures your sensitive user preferences remain secure while maintaining ease of use and performance. 