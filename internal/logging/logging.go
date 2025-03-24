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
	// LDAPLevel logs LDAP-specific debug information (for LDAP protocol debugging)
	LDAPLevel
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
		LDAPLevel:  "LDAP",
		TraceLevel: "TRACE",
	}

	// Current log level
	currentLevel = InfoLevel

	// Logger instances for each level
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	warnLogger  = log.New(os.Stderr, "[WARN]  ", log.LstdFlags)
	infoLogger  = log.New(os.Stdout, "[INFO]  ", log.LstdFlags)
	debugLogger = log.New(os.Stdout, "[DEBUG] ", log.LstdFlags)
	ldapLogger  = log.New(os.Stdout, "[LDAP]  ", log.LstdFlags)
	traceLogger = log.New(os.Stdout, "[TRACE] ", log.LstdFlags)
)

// SetLevel sets the logging level
func SetLevel(level LogLevel) {
	currentLevel = level
}

// GetLevel returns the current logging level
func GetLevel() LogLevel {
	return currentLevel
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
func Error(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}

// Warn logs a warning message if the log level is WarnLevel or higher
func Warn(format string, v ...interface{}) {
	if currentLevel >= WarnLevel {
		warnLogger.Printf(format, v...)
	}
}

// Info logs an informational message if the log level is InfoLevel or higher
func Info(format string, v ...interface{}) {
	if currentLevel >= InfoLevel {
		infoLogger.Printf(format, v...)
	}
}

// Debug logs a debug message if the log level is DebugLevel or higher
func Debug(format string, v ...interface{}) {
	if currentLevel >= DebugLevel {
		debugLogger.Printf(format, v...)
	}
}

// LDAP logs an LDAP-specific message if the log level is LDAPLevel or higher
func LDAP(format string, v ...interface{}) {
	if currentLevel >= LDAPLevel {
		ldapLogger.Printf(format, v...)
	}
}

// Trace logs a trace message if the log level is TraceLevel
func Trace(format string, v ...interface{}) {
	if currentLevel >= TraceLevel {
		traceLogger.Printf(format, v...)
	}
}

// SetOutput sets the output destination for a specific log level
func SetOutput(level LogLevel, w io.Writer) {
	switch level {
	case ErrorLevel:
		errorLogger.SetOutput(w)
	case WarnLevel:
		warnLogger.SetOutput(w)
	case InfoLevel:
		infoLogger.SetOutput(w)
	case DebugLevel:
		debugLogger.SetOutput(w)
	case LDAPLevel:
		ldapLogger.SetOutput(w)
	case TraceLevel:
		traceLogger.SetOutput(w)
	}
}

// SetPrefix sets the prefix for a specific log level
func SetPrefix(level LogLevel, prefix string) {
	switch level {
	case ErrorLevel:
		errorLogger.SetPrefix(prefix)
	case WarnLevel:
		warnLogger.SetPrefix(prefix)
	case InfoLevel:
		infoLogger.SetPrefix(prefix)
	case DebugLevel:
		debugLogger.SetPrefix(prefix)
	case LDAPLevel:
		ldapLogger.SetPrefix(prefix)
	case TraceLevel:
		traceLogger.SetPrefix(prefix)
	}
}

// SetFlags sets the flags for a specific log level
func SetFlags(level LogLevel, flags int) {
	switch level {
	case ErrorLevel:
		errorLogger.SetFlags(flags)
	case WarnLevel:
		warnLogger.SetFlags(flags)
	case InfoLevel:
		infoLogger.SetFlags(flags)
	case DebugLevel:
		debugLogger.SetFlags(flags)
	case LDAPLevel:
		ldapLogger.SetFlags(flags)
	case TraceLevel:
		traceLogger.SetFlags(flags)
	}
}