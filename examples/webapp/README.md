# UserPrefs Web Application Example

This example demonstrates how to use the UserPrefs library in a real-world web application scenario with multiple storage backends, caching, and encryption.

## Overview

This example simulates a complete web application that manages user preferences across different categories:

- **Profile Settings**: Public user information (display name, avatar, bio)
- **Application Settings**: UI preferences (theme, language, notifications)
- **API Integrations**: Encrypted third-party service credentials
- **Account Settings**: User verification status and personal information
- **Billing Settings**: Subscription tiers and rate limits

## Features Demonstrated

1. **Flexible Storage Configuration**: Memory and SQLite storage backends
2. **Optional Caching**: Memory cache for improved performance
3. **Encryption Support**: Automatic encryption for sensitive data
4. **Environment-based Configuration**: Storage and cache selection via environment variables
5. **Real-world Scenarios**: User registration, settings updates, API integrations, bulk operations

## Running the Example

### Basic Usage (Memory Storage)

```bash
# From the webapp example directory
go run main.go
```

### With SQLite Storage

```bash
STORAGE_TYPE=sqlite SQLITE_PATH=webapp.db go run main.go
```

### With Encryption Enabled

```bash
USERPREFS_ENCRYPTION_KEY='your-32-byte-encryption-key-here!!' go run main.go
```

### Full Configuration

```bash
STORAGE_TYPE=sqlite \
SQLITE_PATH=webapp.db \
CACHE_TYPE=memory \
USERPREFS_ENCRYPTION_KEY='your-32-byte-encryption-key-here!!' \
go run main.go
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_TYPE` | `memory` | Storage backend (`memory`, `sqlite`) |
| `SQLITE_PATH` | `webapp_prefs.db` | SQLite database file path |
| `CACHE_TYPE` | `memory` | Cache backend (`memory`, `none`) |
| `USERPREFS_ENCRYPTION_KEY` | (none) | 32+ byte encryption key for sensitive data |

## Scenarios Demonstrated

### 1. New User Registration
- Default preference values
- Initial profile setup
- Account verification
- Category-based preference retrieval

### 2. Existing User Settings Update
- Loading existing preferences
- Partial updates (theme changes)
- Subscription upgrades
- Billing preference management

### 3. API Integration Management
- Storing encrypted API credentials
- Multiple service integrations (GitHub, Slack)
- Automatic encryption/decryption
- Secure credential retrieval

### 4. Bulk Operations
- Setting multiple preferences at once
- Exporting all user preferences
- Category-based preference analysis
- Settings summary and reporting

## Security Features

### Encrypted Preferences

The following preference types are automatically encrypted when an encryption key is provided:

- `api_credentials_github` - GitHub API credentials
- `api_credentials_slack` - Slack API credentials  
- `personal_info` - Personal identifiable information

### Non-Encrypted Preferences

Public or non-sensitive data remains unencrypted for performance:

- `user_profile` - Public profile information
- `app_settings` - UI and application preferences
- `email_verified` - Account verification status
- `subscription_tier` - Billing tier information
- `api_rate_limit` - Rate limiting configuration

## Data Types Supported

- **String**: Simple text values with optional allowed values
- **Boolean**: True/false flags
- **Integer**: Numeric values (rate limits, counts)
- **JSON**: Complex structured data (profiles, settings, credentials)

## Production Considerations

1. **Storage**: Use persistent storage (SQLite, PostgreSQL) in production
2. **Caching**: Enable caching for better performance with high user loads
3. **Encryption**: Always encrypt sensitive data with strong keys
4. **Environment Variables**: Never hardcode encryption keys or database credentials
5. **Error Handling**: Implement proper error handling and logging
6. **Monitoring**: Track preference access patterns and performance metrics

## Example Output

When run, the example will show:

```
UserPrefs Web Application Example
=================================
✓ Using memory storage (development mode)
✓ Using memory cache
ℹ Encryption disabled (no USERPREFS_ENCRYPTION_KEY set)
✓ Defined 8 preference types

==================================================
SCENARIO 1: New User Registration
==================================================
👤 New user registration: user_001
📋 Default app settings: ...
✓ User profile created
✓ Email marked as verified
📊 Account preferences summary: ...

[Additional scenarios follow...]
```

This demonstrates a complete user preference management system suitable for production web applications. 