package userprefs

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestDefaultLogger(t *testing.T) {
	// Redirect log output to a buffer for testing
	var buf bytes.Buffer
	logger := &defaultLogger{
		logger: log.New(&buf, "", 0),
	}

	// Test messages
	logger.Debug("Debug message", "arg1", 123)
	logger.Info("Info message")
	logger.Warn("Warn message", "arg2")
	logger.Error("Error message", "arg3")

	logOutput := buf.String()

	// Check if messages contain correct levels and content
	if !strings.Contains(logOutput, "DEBUG: Debug message [arg1 123]") {
		t.Errorf("Debug message not logged correctly. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "INFO: Info message") {
		t.Errorf("Info message not logged correctly. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "WARN: Warn message [arg2]") {
		t.Errorf("Warn message not logged correctly. Got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "ERROR: Error message [arg3]") {
		t.Errorf("Error message not logged correctly. Got: %s", logOutput)
	}
}

func TestDefaultLoggerOutput(t *testing.T) {
	t.Name()
	// Reset logger to use os.Stderr and ensure no panic occurs
	logger := newDefaultLogger()
	logger.Error("Test error message")
}
