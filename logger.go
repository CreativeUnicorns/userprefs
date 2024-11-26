// logger.go
package userprefs

import (
    "fmt"
    "log"
    "os"
)

type defaultLogger struct {
    logger *log.Logger
}

func newDefaultLogger() Logger {
    return &defaultLogger{
        logger: log.New(os.Stderr, "", log.LstdFlags),
    }
}

func (l *defaultLogger) Debug(msg string, args ...interface{}) {
    l.log("DEBUG", msg, args...)
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
    l.log("INFO", msg, args...)
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
    l.log("WARN", msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
    l.log("ERROR", msg, args...)
}

func (l *defaultLogger) log(level, msg string, args ...interface{}) {
    if len(args) > 0 {
        msg = fmt.Sprintf("%s: %s %v", level, msg, args)
    } else {
        msg = fmt.Sprintf("%s: %s", level, msg)
    }
    l.logger.Println(msg)
}