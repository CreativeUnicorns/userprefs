// Package userprefs provides default logging implementations.
package userprefs

import (
	"log/slog"
	"os"
)

// defaultSlogLogger is an implementation of the Logger interface using the slog package.
type defaultSlogLogger struct {
	slogger *slog.Logger
}

// newDefaultLogger initializes a new defaultSlogLogger instance.
// It defaults to a JSON handler writing to os.Stderr with a nil LogLevel (which means slog.LevelInfo).
func newDefaultLogger() Logger {
	return &defaultSlogLogger{
		slogger: slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo, // Default to Info level, can be made configurable later
		})),
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
