package logger

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger provides secure logging with automatic PII redaction
type Logger struct {
	mu       sync.RWMutex
	level    LogLevel
	logger   *log.Logger
	isDev    bool
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Initialize sets up the default logger instance
func Initialize(level LogLevel, isDev bool) {
	once.Do(func() {
		defaultLogger = &Logger{
			level:  level,
			logger: log.New(os.Stdout, "", log.LstdFlags),
			isDev:  isDev,
		}
	})
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		// Initialize with default settings if not already done
		Initialize(INFO, false)
	}
	return defaultLogger
}

// SetLevel updates the log level
func SetLevel(level LogLevel) {
	if defaultLogger != nil {
		defaultLogger.mu.Lock()
		defaultLogger.level = level
		defaultLogger.mu.Unlock()
	}
}

// redactEmail redacts email addresses for privacy
func redactEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "****"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return "****@" + domain
	}

	return local[0:1] + "****" + local[len(local)-1:] + "@" + domain
}

// hashUserID creates a consistent hash for user IDs
func hashUserID(userID interface{}) string {
	str := fmt.Sprintf("%v", userID)
	hash := sha256.Sum256([]byte(str))
	return fmt.Sprintf("user_%x", hash[:4])
}

// truncateID truncates IDs like session or pack IDs
func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:4] + "****"
}

// redactValue redacts sensitive values based on the key name
func redactValue(key string, value interface{}) interface{} {
	keyLower := strings.ToLower(key)
	valueStr := fmt.Sprintf("%v", value)

	// Email redaction
	if strings.Contains(keyLower, "email") || strings.Contains(valueStr, "@") {
		return redactEmail(valueStr)
	}

	// User ID hashing
	if strings.Contains(keyLower, "userid") || strings.Contains(keyLower, "user_id") {
		return hashUserID(value)
	}

	// Session ID truncation
	if strings.Contains(keyLower, "session") || strings.Contains(keyLower, "token") {
		return truncateID(valueStr)
	}

	// Pack ID truncation
	if strings.Contains(keyLower, "packid") || strings.Contains(keyLower, "pack_id") {
		if len(valueStr) > 8 {
			return truncateID(valueStr)
		}
	}

	// Password complete redaction
	if strings.Contains(keyLower, "password") {
		return "[REDACTED]"
	}

	return value
}

// formatMessage formats a log message with key-value pairs
func (l *Logger) formatMessage(level, msg string, keysAndValues ...interface{}) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[%s] %s", level, msg))

	if len(keysAndValues) > 0 {
		builder.WriteString(" {")
		for i := 0; i < len(keysAndValues); i += 2 {
			if i > 0 {
				builder.WriteString(",")
			}

			key := fmt.Sprintf("%v", keysAndValues[i])
			var value interface{}

			if i+1 < len(keysAndValues) {
				value = keysAndValues[i+1]
			} else {
				value = ""
			}

			// Apply redaction unless in dev mode with DEBUG level
			if !l.isDev || l.level > DEBUG {
				value = redactValue(key, value)
			}

			builder.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
		builder.WriteString(" }")
	}

	return builder.String()
}

// shouldLog checks if a message should be logged based on level
func (l *Logger) shouldLog(level LogLevel) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return level >= l.level
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if l.shouldLog(DEBUG) {
		l.logger.Println(l.formatMessage("DEBUG", msg, keysAndValues...))
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	if l.shouldLog(INFO) {
		l.logger.Println(l.formatMessage("INFO", msg, keysAndValues...))
	}
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	if l.shouldLog(WARN) {
		l.logger.Println(l.formatMessage("WARN", msg, keysAndValues...))
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	if l.shouldLog(ERROR) {
		l.logger.Println(l.formatMessage("ERROR", msg, keysAndValues...))
	}
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(msg string, keysAndValues ...interface{}) {
	GetLogger().Debug(msg, keysAndValues...)
}

// Info logs an info message using the default logger
func Info(msg string, keysAndValues ...interface{}) {
	GetLogger().Info(msg, keysAndValues...)
}

// Warn logs a warning message using the default logger
func Warn(msg string, keysAndValues ...interface{}) {
	GetLogger().Warn(msg, keysAndValues...)
}

// Error logs an error message using the default logger
func Error(msg string, keysAndValues ...interface{}) {
	GetLogger().Error(msg, keysAndValues...)
}

// ParseLevel converts a string to a LogLevel
func ParseLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}