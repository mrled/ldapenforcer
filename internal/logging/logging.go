package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// LogLevel defines the level of logging
type LogLevel int

// Log levels
const (
	// ErrorLevel logs only errors
	ErrorLevel LogLevel = iota
	// WarnLevel logs warnings and errors
	WarnLevel
	// InfoLevel logs informational messages, warnings, and errors
	InfoLevel
	// DebugLevel logs debug messages, informational messages, warnings, and errors
	DebugLevel
	// TraceLevel logs everything including very detailed traces
	TraceLevel
)

var (
	// levelNames maps log levels to their string representations
	levelNames = map[LogLevel]string{
		ErrorLevel: "ERROR",
		WarnLevel:  "WARN",
		InfoLevel:  "INFO",
		DebugLevel: "DEBUG",
		TraceLevel: "TRACE",
	}
)

// Logger represents a configured logger with its own level
type Logger struct {
	currentLevel LogLevel
	errorLogger  *log.Logger
	warnLogger   *log.Logger
	infoLogger   *log.Logger
	debugLogger  *log.Logger
	traceLogger  *log.Logger
}

// Default logger instance for application-wide logging
var DefaultLogger = NewLogger("APP")

// LDAPProtocolLogger is a dedicated logger for LDAP protocol operations
var LDAPProtocolLogger = NewLogger("LDAP")

// NewLogger creates a new logger with a specific name
func NewLogger(name string) *Logger {
	return &Logger{
		currentLevel: ErrorLevel, // Default to ERROR level
		errorLogger:  log.New(os.Stderr, fmt.Sprintf("[%s:ERROR] ", name), log.LstdFlags),
		warnLogger:   log.New(os.Stderr, fmt.Sprintf("[%s:WARN]  ", name), log.LstdFlags),
		infoLogger:   log.New(os.Stdout, fmt.Sprintf("[%s:INFO]  ", name), log.LstdFlags),
		debugLogger:  log.New(os.Stdout, fmt.Sprintf("[%s:DEBUG] ", name), log.LstdFlags),
		traceLogger:  log.New(os.Stdout, fmt.Sprintf("[%s:TRACE] ", name), log.LstdFlags),
	}
}

// SetLevel sets the logging level for this logger
func (l *Logger) SetLevel(level LogLevel) {
	l.currentLevel = level
}

// GetLevel returns the current logging level for this logger
func (l *Logger) GetLevel() LogLevel {
	return l.currentLevel
}

// GetLevelName returns the string representation of a log level
func GetLevelName(level LogLevel) string {
	if name, ok := levelNames[level]; ok {
		return name
	}
	return fmt.Sprintf("Level(%d)", level)
}

// ParseLevel parses a string into a LogLevel
func ParseLevel(level string) (LogLevel, error) {
	level = strings.TrimSpace(level)
	level = strings.ToUpper(level)
	for k, v := range levelNames {
		if v == level {
			return k, nil
		}
	}
	return InfoLevel, fmt.Errorf("invalid log level: %s", level)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

// Warn logs a warning message if the log level is WarnLevel or higher
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.currentLevel >= WarnLevel {
		l.warnLogger.Printf(format, v...)
	}
}

// Info logs an informational message if the log level is InfoLevel or higher
func (l *Logger) Info(format string, v ...interface{}) {
	if l.currentLevel >= InfoLevel {
		l.infoLogger.Printf(format, v...)
	}
}

// Debug logs a debug message if the log level is DebugLevel or higher
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.currentLevel >= DebugLevel {
		l.debugLogger.Printf(format, v...)
	}
}

// Trace logs a trace message if the log level is TraceLevel
func (l *Logger) Trace(format string, v ...interface{}) {
	if l.currentLevel >= TraceLevel {
		l.traceLogger.Printf(format, v...)
	}
}

// SetOutput sets the output destination for a specific log level
func (l *Logger) SetOutput(level LogLevel, w io.Writer) {
	switch level {
	case ErrorLevel:
		l.errorLogger.SetOutput(w)
	case WarnLevel:
		l.warnLogger.SetOutput(w)
	case InfoLevel:
		l.infoLogger.SetOutput(w)
	case DebugLevel:
		l.debugLogger.SetOutput(w)
	case TraceLevel:
		l.traceLogger.SetOutput(w)
	}
}

// SetPrefix sets the prefix for a specific log level
func (l *Logger) SetPrefix(level LogLevel, prefix string) {
	switch level {
	case ErrorLevel:
		l.errorLogger.SetPrefix(prefix)
	case WarnLevel:
		l.warnLogger.SetPrefix(prefix)
	case InfoLevel:
		l.infoLogger.SetPrefix(prefix)
	case DebugLevel:
		l.debugLogger.SetPrefix(prefix)
	case TraceLevel:
		l.traceLogger.SetPrefix(prefix)
	}
}

// SetFlags sets the flags for a specific log level
func (l *Logger) SetFlags(level LogLevel, flags int) {
	switch level {
	case ErrorLevel:
		l.errorLogger.SetFlags(flags)
	case WarnLevel:
		l.warnLogger.SetFlags(flags)
	case InfoLevel:
		l.infoLogger.SetFlags(flags)
	case DebugLevel:
		l.debugLogger.SetFlags(flags)
	case TraceLevel:
		l.traceLogger.SetFlags(flags)
	}
}
