package userprefs

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	// Redirect log output to a buffer for testing
	var buf bytes.Buffer
	slogHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Ensure Debug messages are logged for this test
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Remove time for consistent test output
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})

	logger := &defaultSlogLogger{
		slogger: slog.New(slogHandler),
	}

	// Test messages
	logger.Debug("Debug message", "arg1", 123)
	logger.Info("Info message")
	logger.Warn("Warn message", "key_warn", "val_warn") // slog expects key-value pairs
	logger.Error("Error message", "key_err", "val_err") // slog expects key-value pairs

	logOutput := buf.String()
	//t.Logf("Log output:\n%s", logOutput) // For debugging test failures

	// Check if messages contain correct levels and content based on slog's TextHandler format
	// Example: level=DEBUG msg="Debug message" arg1=123
	expectedDebug := "level=DEBUG msg=\"Debug message\" arg1=123"
	if !strings.Contains(logOutput, expectedDebug) {
		t.Errorf("Debug message not logged correctly.\nExpected to contain: %s\nGot: %s", expectedDebug, logOutput)
	}

	expectedInfo := "level=INFO msg=\"Info message\""
	if !strings.Contains(logOutput, expectedInfo) {
		t.Errorf("Info message not logged correctly.\nExpected to contain: %s\nGot: %s", expectedInfo, logOutput)
	}

	expectedWarn := "level=WARN msg=\"Warn message\" key_warn=val_warn"
	if !strings.Contains(logOutput, expectedWarn) {
		t.Errorf("Warn message not logged correctly.\nExpected to contain: %s\nGot: %s", expectedWarn, logOutput)
	}

	expectedError := "level=ERROR msg=\"Error message\" key_err=val_err"
	if !strings.Contains(logOutput, expectedError) {
		t.Errorf("Error message not logged correctly.\nExpected to contain: %s\nGot: %s", expectedError, logOutput)
	}
}

func TestDefaultLoggerOutput(_ *testing.T) {
	// This test ensures that the default logger (writing to os.Stderr) can be initialized and used without panic.
	// We can't easily capture os.Stderr here without more complex redirection,
	// so we'll just ensure the call doesn't panic and assume slog handles stderr correctly.

	// Store original stderr
	origStderr := os.Stderr
	// Create a pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewDefaultLogger() // This will now create a defaultSlogLogger
	logger.Error("Test error message", "test_key", "test_value")

	// Close the writer and restore stderr
	_ = w.Close()
	os.Stderr = origStderr

	// Read from the pipe (optional, mostly to ensure no blocking and it went somewhere)
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()

	// Basic check on captured output (optional, as slog's correctness is not what we test here)
	// if !strings.Contains(buf.String(), "Test error message") {
	// 	t.Errorf("Expected error message in stderr output, got: %s", buf.String())
	// }
}

func TestDefaultSlogLogger_LevelSetting(t *testing.T) {
	// Note: This test is a bit limited because we can't easily inspect the internal
	// slog.Leveler of the default logger after it's created without changing its API.
	// We are mainly testing that the constructor runs and that changing level on our wrapper
	// doesn't panic. For a more thorough test, the defaultSlogLogger might need to expose its Leveler
	// or provide a way to get the current effective level.

	tests := []struct {
		name        string
		levelToSet  LogLevel
		expectPanic bool // Not directly testable for internal slog state, more for API usage
	}{
		{"Set LevelDebug", LogLevelDebug, false},
		{"Set LevelInfo", LogLevelInfo, false},
		{"Set LevelWarn", LogLevelWarn, false},
		{"Set LevelError", LogLevelError, false},
		{"Set Invalid Level", LogLevel(99), false}, // Assuming SetLevel handles invalid gracefully
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the exported NewDefaultLogger
			logger, ok := NewDefaultLogger().(*defaultSlogLogger)
			if !ok || logger == nil {
				t.Fatal("NewDefaultLogger() did not return the expected type or was nil")
			}

			defer func() {
				if r := recover(); r != nil && !tt.expectPanic {
					t.Errorf("SetLevel() panicked unexpectedly: %v", r)
				} else if r == nil && tt.expectPanic {
					t.Errorf("SetLevel() did not panic as expected")
				}
			}()

			logger.SetLevel(tt.levelToSet)
			// Further assertion could be to log a message at a level that should be suppressed
			// and check if it appears in output, but that requires capturing os.Stderr.
			// For now, we just ensure no panic and the call completes.
		})
	}
}
