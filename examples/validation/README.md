# UserPrefs Validation Example

This example demonstrates the `ValidateFunc` feature of the userprefs library, which allows you to add custom validation logic to preference definitions beyond basic type checking and `AllowedValues` constraints.

## Overview

The `ValidateFunc` field in `PreferenceDefinition` accepts a function with the signature:

```go
func(value interface{}) error
```

This function is called during `Set` operations after basic type validation and `AllowedValues` checking (if configured). If the function returns an error, the preference value is rejected and not stored.

## Key Features

- **Type Safety**: The value passed to `ValidateFunc` has already been validated for the correct type
- **Error Handling**: Return any error to reject the value with a descriptive message
- **Flexible Logic**: Implement any custom validation logic your application needs
- **Performance**: Validation only runs during `Set` operations, not during `Get`

## Validation Examples

### 1. Range Validation (Integers)

```go
{
    Key:          "page_size",
    Type:         userprefs.IntType,
    DefaultValue: 20,
    ValidateFunc: func(value interface{}) error {
        pageSize := value.(int) // Type is already validated
        if pageSize < 1 || pageSize > 100 {
            return fmt.Errorf("page size must be between 1 and 100, got %d", pageSize)
        }
        return nil
    },
}
```

### 2. Range Validation (Floats)

```go
{
    Key:          "volume_level",
    Type:         userprefs.FloatType,
    DefaultValue: 0.5,
    ValidateFunc: func(value interface{}) error {
        volume := value.(float64) // Type is already validated
        if volume < 0.0 || volume > 1.0 {
            return fmt.Errorf("volume level must be between 0.0 and 1.0, got %.2f", volume)
        }
        return nil
    },
}
```

### 3. Format Validation (Email)

```go
{
    Key:          "notification_email",
    Type:         userprefs.StringType,
    DefaultValue: "",
    ValidateFunc: func(value interface{}) error {
        email := value.(string)
        if email == "" {
            return nil // Empty email is allowed
        }
        
        emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
        if !emailRegex.MatchString(email) {
            return fmt.Errorf("invalid email format: %s", email)
        }
        return nil
    },
}
```

### 4. Length and Character Validation

```go
{
    Key:          "username",
    Type:         userprefs.StringType,
    DefaultValue: "",
    ValidateFunc: func(value interface{}) error {
        username := value.(string)
        if len(username) < 3 {
            return fmt.Errorf("username must be at least 3 characters long")
        }
        if len(username) > 20 {
            return fmt.Errorf("username must be no more than 20 characters long")
        }
        
        usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
        if !usernameRegex.MatchString(username) {
            return fmt.Errorf("username can only contain letters, numbers, and underscores")
        }
        return nil
    },
}
```

### 5. Complex JSON Validation

```go
{
    Key:          "theme_colors",
    Type:         userprefs.JSONType,
    DefaultValue: map[string]interface{}{"primary": "#007bff", "secondary": "#6c757d"},
    ValidateFunc: func(value interface{}) error {
        colors, ok := value.(map[string]interface{})
        if !ok {
            return fmt.Errorf("theme colors must be a JSON object")
        }
        
        // Ensure required color keys exist
        requiredColors := []string{"primary", "secondary"}
        for _, colorKey := range requiredColors {
            if _, exists := colors[colorKey]; !exists {
                return fmt.Errorf("missing required color: %s", colorKey)
            }
        }
        
        // Validate hex color format
        hexColorRegex := regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
        for colorKey, colorValue := range colors {
            colorStr, ok := colorValue.(string)
            if !ok {
                return fmt.Errorf("color value for %s must be a string", colorKey)
            }
            if !hexColorRegex.MatchString(colorStr) {
                return fmt.Errorf("invalid hex color format for %s: %s", colorKey, colorStr)
            }
        }
        return nil
    },
}
```

### 6. Business Logic Validation

```go
{
    Key:          "api_endpoint",
    Type:         userprefs.StringType,
    DefaultValue: "https://api.example.com",
    ValidateFunc: func(value interface{}) error {
        endpoint := value.(string)
        
        // Must be a valid URL
        if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
            return fmt.Errorf("API endpoint must start with http:// or https://")
        }
        
        // Must not be localhost in production (example business rule)
        if strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1") {
            return fmt.Errorf("localhost endpoints are not allowed in production")
        }
        
        return nil
    },
}
```

## Running the Example

```bash
cd examples/validation
go run main.go
```

This will run through all validation scenarios and show which values are accepted and which are rejected.

## Best Practices

1. **Keep validation functions simple and focused** - Each function should validate one specific aspect
2. **Provide clear error messages** - Users should understand why their input was rejected
3. **Consider performance** - Validation runs on every `Set` operation
4. **Handle edge cases** - Consider empty values, nil values, and boundary conditions
5. **Use type assertions safely** - The type is already validated, but be explicit about it
6. **Combine with AllowedValues** - Use `AllowedValues` for simple enumeration, `ValidateFunc` for complex logic

## Error Handling

When validation fails, the error is wrapped with `ErrInvalidValue` and includes the custom error message:

```go
err := mgr.Set(ctx, userID, "page_size", 150)
// err will be: invalid preference value: custom validation failed: page size must be between 1 and 100, got 150

if errors.Is(err, userprefs.ErrInvalidValue) {
    // Handle validation error
}
```

## Integration with Other Features

- **Type Validation**: Runs before `ValidateFunc`
- **AllowedValues**: Checked before `ValidateFunc`
- **Encryption**: Applied after successful validation
- **Caching**: Only valid values are cached
- **Storage**: Only valid values are persisted 