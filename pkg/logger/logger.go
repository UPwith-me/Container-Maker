package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger is a simple logger
type Logger struct {
	level  Level
	output io.Writer
	file   *os.File
}

var defaultLogger = &Logger{
	level:  LevelInfo,
	output: os.Stderr,
}

// Init initializes the logger with optional file output
func Init(logToFile bool) error {
	if logToFile {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		logDir := filepath.Join(home, ".cm", "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		logFile := filepath.Join(logDir, fmt.Sprintf("cm-%s.log", time.Now().Format("2006-01-02")))
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}

		defaultLogger.file = f
		defaultLogger.output = io.MultiWriter(os.Stderr, f)
	}
	return nil
}

// Close closes the log file
func Close() {
	if defaultLogger.file != nil {
		defaultLogger.file.Close()
	}
}

// SetLevel sets the log level
func SetLevel(level Level) {
	defaultLogger.level = level
}

// SetLevelFromString sets log level from string
func SetLevelFromString(level string) {
	switch level {
	case "debug":
		defaultLogger.level = LevelDebug
	case "info":
		defaultLogger.level = LevelInfo
	case "warn":
		defaultLogger.level = LevelWarn
	case "error":
		defaultLogger.level = LevelError
	}
}

func log(level Level, prefix, format string, args ...interface{}) {
	if level < defaultLogger.level {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(defaultLogger.output, "[%s] %s %s\n", timestamp, prefix, msg)
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	log(LevelDebug, "DEBUG", format, args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	log(LevelInfo, "INFO ", format, args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	log(LevelWarn, "WARN ", format, args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	log(LevelError, "ERROR", format, args...)
}

// Debugf is an alias for Debug
func Debugf(format string, args ...interface{}) {
	Debug(format, args...)
}

// Infof is an alias for Info
func Infof(format string, args ...interface{}) {
	Info(format, args...)
}

// Warnf is an alias for Warn
func Warnf(format string, args ...interface{}) {
	Warn(format, args...)
}

// Errorf is an alias for Error
func Errorf(format string, args ...interface{}) {
	Error(format, args...)
}
