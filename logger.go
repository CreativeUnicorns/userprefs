// Package userprefs provides default logging implementations.
package userprefs

import (
	"log/slog"
	"os"
)

// LogLevel defines the various log levels.
// These correspond to slog's levels.
// Using a custom type allows for clearer API contracts within the userprefs package.
type LogLevel int

// Log level constants, mirroring slog levels for internal mapping.
const (
	LogLevelDebug LogLevel = LogLevel(slog.LevelDebug) // Debug messages
	LogLevelInfo  LogLevel = LogLevel(slog.LevelInfo)  // Informational messages
	LogLevelWarn  LogLevel = LogLevel(slog.LevelWarn)  // Warning messages
	LogLevelError LogLevel = LogLevel(slog.LevelError) // Error messages
)

// Logger defines the interface for logging operations.
// This allows for different logging implementations to be used.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	SetLevel(level LogLevel) // New method to set the log level dynamically
}

// defaultSlogLogger is an implementation of the Logger interface using the slog package.
type defaultSlogLogger struct {
	slogger  *slog.Logger
	levelVar *slog.LevelVar // To control the log level dynamically
}

// NewDefaultLogger initializes a new defaultSlogLogger instance.
// It defaults to a JSON handler writing to os.Stderr with slog.LevelInfo.
// The log level can be changed dynamically via the SetLevel method.
// This function is now exported.
func NewDefaultLogger() Logger {
	levelVar := new(slog.LevelVar) // Create a new LevelVar
	levelVar.Set(slog.LevelInfo)   // Default to Info level

	handlerOpts := &slog.HandlerOptions{
		Level: levelVar, // Use the LevelVar here
	}
	return &defaultSlogLogger{
		slogger:  slog.New(slog.NewJSONHandler(os.Stderr, handlerOpts)),
		levelVar: levelVar,
	}
}

// Debug logs a debug-level message.
func (l *defaultSlogLogger) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

// Info logs an info-level message.
func (l *defaultSlogLogger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// Warn logs a warning-level message.
func (l *defaultSlogLogger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

// Error logs an error-level message.
func (l *defaultSlogLogger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// SetLevel changes the logging level of the defaultSlogLogger dynamically.
func (l *defaultSlogLogger) SetLevel(level LogLevel) {
	if l.levelVar != nil {
		l.levelVar.Set(slog.Level(level)) // Convert our LogLevel back to slog.Level
	}
}
