# Supabase Integration Example

This example demonstrates how to use the userprefs library with Supabase as the PostgreSQL database provider. Supabase provides a powerful, open-source Firebase alternative with real-time subscriptions, instant APIs, and edge functions.

## Features Demonstrated

- ✅ **Supabase PostgreSQL Integration**: Direct connection to Supabase's managed PostgreSQL database
- ✅ **Environment Variable Configuration**: Secure handling of credentials using .env files
- ✅ **Complex JSON Preferences**: Nested structures for user profiles and project settings
- ✅ **Performance Optimization**: Optional Redis caching for improved response times
- ✅ **Concurrent Operations**: Real-time preference updates from multiple sessions
- ✅ **Category-based Organization**: Logical grouping of related preferences
- ✅ **Type Safety**: Strongly-typed preference structures with JSON marshaling
- ✅ **HTTP API Server**: RESTful API for preference management
- ✅ **API Client Example**: Demonstrates HTTP client usage
- ✅ **Comprehensive Testing**: Integration tests for all components

## Prerequisites

1. **Supabase Project**: Create a free account at [supabase.com](https://supabase.com)
2. **Go 1.24.3+**: Make sure you have Go installed
3. **Optional: Redis**: For enhanced caching performance

## Quick Start

### 1. Set up Supabase

1. Create a new project at [app.supabase.com](https://app.supabase.com)
2. Go to Settings → Database and note your connection string
3. Go to Settings → API and copy your project URL and anon key

### 2. Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your Supabase credentials
```

Update `.env` with your actual Supabase values:

```env
# From your Supabase project dashboard
SUPABASE_URL=https://your-project-ref.supabase.co
SUPABASE_ANON_KEY=your-anon-key-here
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key-here

# Database connection (replace YOUR-PASSWORD and YOUR-PROJECT-REF)
SUPABASE_DB_URL=postgres://postgres:your-db-password@db.your-project-ref.supabase.co:5432/postgres

# Optional: Redis for caching
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### 3. Initialize Database Schema

**Important**: You need to run the database initialization script in your Supabase database before running the examples.

#### Option A: Using Supabase Dashboard (Recommended)
1. Go to your Supabase project dashboard
2. Navigate to the SQL Editor
3. Copy the contents of `init.sql` and paste it into a new query
4. Click "Run" to execute the script

#### Option B: Using psql (if you have PostgreSQL client installed)
```bash
# Replace with your actual database URL
PGPASSWORD="your-db-password" psql -h db.your-project-ref.supabase.co -p 5432 -U postgres -d postgres -f init.sql
```

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Run the Example

There are several ways to run the examples:

#### CLI Demo (recommended for first try)
```bash
# With local Docker environment
make run-local

# With Supabase cloud (requires .env configuration)
make run
```

#### HTTP API Server
```bash
# Start API server with local Docker
make run-api-local

# Start API server with Supabase cloud
make run-api

# In another terminal, test the API
make run-client
```

#### Manual execution
```bash
# CLI demo
go run main.go

# API server
go run cmd/api/api_server.go

# API client (requires API server running)
go run cmd/client/client_example.go
```

## What the Example Does

The Supabase example includes three main components:

### 1. **CLI Demo** (`main.go`)
A comprehensive command-line demonstration that shows:
```go
type UserProfile struct {
    Theme         string `json:"theme"`
    Language      string `json:"language"`
    Timezone      string `json:"timezone"`
    EmailSettings struct {
        Marketing bool `json:"marketing"`
        Security  bool `json:"security"`
        Updates   bool `json:"updates"`
    } `json:"email_settings"`
    UIPreferences struct {
        SidebarCollapsed bool   `json:"sidebar_collapsed"`
        DensityMode      string `json:"density_mode"`
        FontSize         int    `json:"font_size"`
    } `json:"ui_preferences"`
}
```

### 2. **Project Settings**
```go
type ProjectSettings struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    IsPublic    bool     `json:"is_public"`
    Tags        []string `json:"tags"`
    Settings    struct {
        AutoSave         bool `json:"auto_save"`
        AutoSaveInterval int  `json:"auto_save_interval"`
        MaxFileSize      int  `json:"max_file_size"`
    } `json:"settings"`
}
```

### 3. **Demonstration Scenarios**

1. **New User Defaults**: Shows how new users get sensible default preferences
2. **Preference Customization**: Demonstrates updating complex nested preferences
3. **Multi-User Management**: Simulates different users with different settings
4. **Bulk Operations**: Retrieves and analyzes preferences across categories
5. **Performance Testing**: Measures cache performance with 100 reads
6. **Concurrent Updates**: Simulates real-time updates from multiple sessions

## Supabase Database Schema

The userprefs library automatically creates the required tables in your Supabase database:

```sql
-- Preference definitions table
CREATE TABLE preference_definitions (
    key TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    category TEXT NOT NULL,
    default_value JSONB,
    allowed_values JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- User preferences table
CREATE TABLE user_preferences (
    user_id TEXT NOT NULL,
    preference_key TEXT NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (user_id, preference_key),
    FOREIGN KEY (preference_key) REFERENCES preference_definitions(key)
);
```

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `SUPABASE_URL` | ✅ | Your Supabase project URL |
| `SUPABASE_ANON_KEY` | ✅ | Anon/public key for client-side access |
| `SUPABASE_SERVICE_ROLE_KEY` | ❌ | Service role key for admin operations |
| `SUPABASE_DB_URL` | ✅ | Direct PostgreSQL connection string |
| `REDIS_URL` | ❌ | Redis URL for caching (optional) |
| `REDIS_PASSWORD` | ❌ | Redis password if required |
| `REDIS_DB` | ❌ | Redis database number (default: 0) |
| `APP_PORT` | ❌ | Application port (default: 8080) |
| `LOG_LEVEL` | ❌ | Logging level (default: info) |

## Performance Considerations

### Caching Strategy
- **Redis**: Best performance for production environments
- **Memory**: Good for development and single-instance deployments
- **No Cache**: Direct database access (not recommended for production)

### Connection Pooling
The example uses Supabase's managed PostgreSQL with automatic connection pooling. For high-traffic applications, consider:
- Adjusting `MaxOpenConns` and `MaxIdleConns` in the storage configuration
- Using Supabase's connection pooler (PgBouncer)
- Implementing read replicas for read-heavy workloads

## Security Best Practices

1. **Environment Variables**: Never commit `.env` files to version control
2. **Row Level Security**: Enable RLS in Supabase for multi-tenant applications
3. **API Keys**: Use anon key for client-side, service role key for server-side only
4. **Database Access**: Restrict database access to specific IP ranges if possible

## Extending the Example

### Adding Real-time Updates
```go
// Subscribe to preference changes using Supabase realtime
supabase := createClient(supabaseURL, supabaseAnonKey)
channel := supabase.Channel("preference-changes")
channel.On("UPDATE", "user_preferences", func(payload map[string]interface{}) {
    // Handle real-time preference updates
})
```

### Adding API Layer
```go
// Add HTTP endpoints for preference management
http.HandleFunc("/api/preferences", handlePreferences)
http.HandleFunc("/api/preferences/bulk", handleBulkPreferences)
```

### Custom Validation
```go
// Add custom validation rules
mgr.AddValidator("user_profile", func(value interface{}) error {
    profile := value.(UserProfile)
    if profile.UIPreferences.FontSize < 8 || profile.UIPreferences.FontSize > 72 {
        return errors.New("font size must be between 8 and 72")
    }
    return nil
})
```

## Testing

### Quick Test
```bash
# Run basic tests
make test

# Run comprehensive integration tests
make integration-test
```

### Manual Testing
```bash
# Test with local Docker environment
make run-local

# Test API server and client
make run-api-local
# In another terminal:
make run-client
```

## Available Make Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make setup` | Install dependencies |
| `make run` | Run CLI demo with Supabase cloud |
| `make run-local` | Run CLI demo with local Docker |
| `make run-api` | Start API server with Supabase cloud |
| `make run-api-local` | Start API server with local Docker |
| `make run-client` | Run API client example |
| `make test` | Run unit tests |
| `make integration-test` | Run comprehensive integration tests |
| `make docker-up` | Start local PostgreSQL and Redis |
| `make docker-down` | Stop local services |
| `make clean` | Clean up generated files |

## Troubleshooting

### Common Issues

1. **Connection Failed**: Check your `SUPABASE_DB_URL` format and credentials
2. **Permission Denied**: Ensure your database user has CREATE and INSERT permissions
3. **SSL Errors**: Add `?sslmode=require` to your connection string if needed
4. **Cache Errors**: Verify Redis connection or fallback to memory cache

### Debug Mode
```bash
export LOG_LEVEL=debug
go run main.go
```

## Related Examples

- [`basic/`](../basic/): Simple SQLite-based example
- [`advanced/`](../advanced/): PostgreSQL + Redis with complex scenarios
- [`sqlite-advanced/`](../sqlite-advanced/): Advanced SQLite features

## Resources

- [Supabase Documentation](https://supabase.com/docs)
- [Supabase Go Client](https://github.com/supabase-community/supabase-go)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [userprefs Library Documentation](../../README.md)
