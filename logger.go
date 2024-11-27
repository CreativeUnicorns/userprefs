// Package userprefs provides default logging implementations.
package userprefs

import (
	"fmt"
	"log"
	"os"
)

// defaultLogger is the default implementation of the Logger interface using the standard log package.
type defaultLogger struct {
	logger *log.Logger
}

// newDefaultLogger initializes a new defaultLogger instance.
func newDefaultLogger() Logger {
	return &defaultLogger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}

// Debug logs a debug-level message.
func (l *defaultLogger) Debug(msg string, args ...interface{}) {
	l.log("DEBUG", msg, args...)
}

// Info logs an info-level message.
func (l *defaultLogger) Info(msg string, args ...interface{}) {
	l.log("INFO", msg, args...)
}

// Warn logs a warning-level message.
func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	l.log("WARN", msg, args...)
}

// Error logs an error-level message.
func (l *defaultLogger) Error(msg string, args ...interface{}) {
	l.log("ERROR", msg, args...)
}

// log is a helper method to format and output log messages.
func (l *defaultLogger) log(level, msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf("%s: %s %v", level, msg, args)
	} else {
		msg = fmt.Sprintf("%s: %s", level, msg)
	}
	l.logger.Println(msg)
}
