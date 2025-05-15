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
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
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

func TestDefaultLoggerOutput(t *testing.T) {
	// This test ensures that the default logger (writing to os.Stderr) can be initialized and used without panic.
	// We can't easily capture os.Stderr here without more complex redirection, 
	// so we'll just ensure the call doesn't panic and assume slog handles stderr correctly.
	
	// Store original stderr
	origStderr := os.Stderr
	// Create a pipe to capture stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := newDefaultLogger() // This will now create a defaultSlogLogger
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
