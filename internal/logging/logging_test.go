package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("TEST")
	if logger.currentLevel != ErrorLevel {
		t.Errorf("Expected new logger to have ERROR level, got %v", logger.currentLevel)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	logger := NewLogger("TEST")

	// Test setting and getting level
	levels := []LogLevel{
		ErrorLevel,
		WarnLevel,
		InfoLevel,
		DebugLevel,
		TraceLevel,
	}

	for _, level := range levels {
		logger.SetLevel(level)
		if logger.GetLevel() != level {
			t.Errorf("Expected level %v, got %v", level, logger.GetLevel())
		}
	}
}

func TestGetLevelName(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{ErrorLevel, "ERROR"},
		{WarnLevel, "WARN"},
		{InfoLevel, "INFO"},
		{DebugLevel, "DEBUG"},
		{TraceLevel, "TRACE"},
		{LogLevel(99), "Level(99)"}, // Invalid level
	}

	for _, tt := range tests {
		if name := GetLevelName(tt.level); name != tt.expected {
			t.Errorf("Expected level name %s, got %s", tt.expected, name)
		}
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input       string
		expected    LogLevel
		expectError bool
	}{
		{"ERROR", ErrorLevel, false},
		{"WARN", WarnLevel, false},
		{"INFO", InfoLevel, false},
		{"DEBUG", DebugLevel, false},
		{"TRACE", TraceLevel, false},
		{"error", ErrorLevel, false},     // Case insensitive
		{"wArN", WarnLevel, false},       // Mixed case
		{"INVALID", InfoLevel, true},     // Invalid level, defaults to INFO with error
		{"", InfoLevel, true},            // Empty string, defaults to INFO with error
		{"  DEBUG  ", DebugLevel, false}, // Whitespace is trimmed
	}

	for _, tt := range tests {
		level, err := ParseLevel(tt.input)
		if tt.expectError && err == nil {
			t.Errorf("Expected error parsing '%s' but got none", tt.input)
		}
		if !tt.expectError && err != nil {
			t.Errorf("Did not expect error parsing '%s' but got: %v", tt.input, err)
		}
		if level != tt.expected {
			t.Errorf("Expected level %v for input '%s', got %v", tt.expected, tt.input, level)
		}
	}
}

func TestLoggerOutputs(t *testing.T) {
	// Create a test logger
	logger := NewLogger("TEST")

	// Create buffers to capture output
	var errorBuf, warnBuf, infoBuf, debugBuf, traceBuf bytes.Buffer

	// Redirect logger outputs
	logger.errorLogger.SetOutput(&errorBuf)
	logger.warnLogger.SetOutput(&warnBuf)
	logger.infoLogger.SetOutput(&infoBuf)
	logger.debugLogger.SetOutput(&debugBuf)
	logger.traceLogger.SetOutput(&traceBuf)

	// Test messages
	errorMsg := "Error message"
	warnMsg := "Warning message"
	infoMsg := "Info message"
	debugMsg := "Debug message"
	traceMsg := "Trace message"

	// Test at ERROR level
	logger.SetLevel(ErrorLevel)
	logger.Error(errorMsg)
	logger.Warn(warnMsg)
	logger.Info(infoMsg)
	logger.Debug(debugMsg)
	logger.Trace(traceMsg)

	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("ERROR level should log error messages")
	}
	if warnBuf.Len() > 0 || infoBuf.Len() > 0 || debugBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("ERROR level should not log warn, info, debug, or trace messages")
	}

	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	traceBuf.Reset()

	// Test at WARN level
	logger.SetLevel(WarnLevel)
	logger.Error(errorMsg)
	logger.Warn(warnMsg)
	logger.Info(infoMsg)
	logger.Debug(debugMsg)
	logger.Trace(traceMsg)

	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("WARN level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("WARN level should log warning messages")
	}
	if infoBuf.Len() > 0 || debugBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("WARN level should not log info, debug, or trace messages")
	}

	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	traceBuf.Reset()

	// Test at INFO level
	logger.SetLevel(InfoLevel)
	logger.Error(errorMsg)
	logger.Warn(warnMsg)
	logger.Info(infoMsg)
	logger.Debug(debugMsg)
	logger.Trace(traceMsg)

	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("INFO level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("INFO level should log warning messages")
	}
	if !strings.Contains(infoBuf.String(), infoMsg) {
		t.Errorf("INFO level should log info messages")
	}
	if debugBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("INFO level should not log debug or trace messages")
	}

	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	traceBuf.Reset()

	// Test at DEBUG level
	logger.SetLevel(DebugLevel)
	logger.Error(errorMsg)
	logger.Warn(warnMsg)
	logger.Info(infoMsg)
	logger.Debug(debugMsg)
	logger.Trace(traceMsg)

	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("DEBUG level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("DEBUG level should log warning messages")
	}
	if !strings.Contains(infoBuf.String(), infoMsg) {
		t.Errorf("DEBUG level should log info messages")
	}
	if !strings.Contains(debugBuf.String(), debugMsg) {
		t.Errorf("DEBUG level should log debug messages")
	}
	if traceBuf.Len() > 0 {
		t.Errorf("DEBUG level should not log trace messages")
	}

	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	traceBuf.Reset()

	// Test at TRACE level
	logger.SetLevel(TraceLevel)
	logger.Error(errorMsg)
	logger.Warn(warnMsg)
	logger.Info(infoMsg)
	logger.Debug(debugMsg)
	logger.Trace(traceMsg)

	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("TRACE level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("TRACE level should log warning messages")
	}
	if !strings.Contains(infoBuf.String(), infoMsg) {
		t.Errorf("TRACE level should log info messages")
	}
	if !strings.Contains(debugBuf.String(), debugMsg) {
		t.Errorf("TRACE level should log debug messages")
	}
	if !strings.Contains(traceBuf.String(), traceMsg) {
		t.Errorf("TRACE level should log trace messages")
	}
}

func TestLoggerSetOutput(t *testing.T) {
	// Create a test logger
	logger := NewLogger("TEST")

	// Create buffer and test redirection
	var buf bytes.Buffer

	// Test setting output for each level
	levels := []LogLevel{
		ErrorLevel,
		WarnLevel,
		InfoLevel,
		DebugLevel,
		TraceLevel,
	}

	messages := map[LogLevel]string{
		ErrorLevel: "Error test",
		WarnLevel:  "Warn test",
		InfoLevel:  "Info test",
		DebugLevel: "Debug test",
		TraceLevel: "Trace test",
	}

	// Set to highest level so all messages are logged
	logger.SetLevel(TraceLevel)

	for _, level := range levels {
		// Reset buffer
		buf.Reset()

		// Set output for this level only
		logger.SetOutput(level, &buf)

		// Log a message at this level
		switch level {
		case ErrorLevel:
			logger.Error(messages[level])
		case WarnLevel:
			logger.Warn(messages[level])
		case InfoLevel:
			logger.Info(messages[level])
		case DebugLevel:
			logger.Debug(messages[level])
		case TraceLevel:
			logger.Trace(messages[level])
		}

		// Check that message was logged to buffer
		if !strings.Contains(buf.String(), messages[level]) {
			t.Errorf("Message for level %s not logged to buffer", GetLevelName(level))
		}

		// Reset output to original
		switch level {
		case ErrorLevel:
			logger.errorLogger.SetOutput(os.Stderr)
		case WarnLevel:
			logger.warnLogger.SetOutput(os.Stderr)
		case InfoLevel:
			logger.infoLogger.SetOutput(os.Stdout)
		case DebugLevel:
			logger.debugLogger.SetOutput(os.Stdout)
		case TraceLevel:
			logger.traceLogger.SetOutput(os.Stdout)
		}
	}
}

func TestLoggerSetPrefix(t *testing.T) {
	// Create a test logger
	logger := NewLogger("TEST")

	// Create buffer to capture output
	var buf bytes.Buffer

	// Set level to error for testing
	logger.SetLevel(ErrorLevel)

	// Set output to buffer
	logger.errorLogger.SetOutput(&buf)

	// Test setting prefix
	testPrefix := "[TEST-PREFIX] "
	logger.SetPrefix(ErrorLevel, testPrefix)

	if logger.errorLogger.Prefix() != testPrefix {
		t.Errorf("Expected prefix '%s', got '%s'", testPrefix, logger.errorLogger.Prefix())
	}

	// Log a message
	logger.Error("Test message")

	// Check that message was logged with new prefix
	if !strings.Contains(buf.String(), testPrefix) {
		t.Errorf("Message not logged with expected prefix '%s'", testPrefix)
	}

	// Restore original settings
	logger.errorLogger.SetOutput(os.Stderr)
}

func TestLoggerSetFlags(t *testing.T) {
	// Create a test logger
	logger := NewLogger("TEST")

	// Create buffer to capture output
	var buf bytes.Buffer

	// Store original flags
	originalFlags := logger.errorLogger.Flags()

	// Set level to error for testing
	logger.SetLevel(ErrorLevel)

	// Set output to buffer
	logger.errorLogger.SetOutput(&buf)

	// Test setting flags
	testFlags := 0 // No flags
	logger.SetFlags(ErrorLevel, testFlags)

	if logger.errorLogger.Flags() != testFlags {
		t.Errorf("Expected flags %d, got %d", testFlags, logger.errorLogger.Flags())
	}

	// Log a message
	logger.Error("Test message")

	// Just check that the message was logged
	if !strings.Contains(buf.String(), "Test message") {
		t.Errorf("Message not logged with flags=0")
	}

	// Restore original settings
	logger.SetFlags(ErrorLevel, originalFlags)
	logger.errorLogger.SetOutput(os.Stderr)
}

func TestDefaultLogger(t *testing.T) {
	// Create buffers to capture output
	var buf bytes.Buffer

	// Store original settings
	originalLevel := DefaultLogger.GetLevel()
	defer DefaultLogger.SetLevel(originalLevel)

	// Test that we can use the global functions that operate on DefaultLogger
	DefaultLogger.SetLevel(ErrorLevel)
	DefaultLogger.SetOutput(ErrorLevel, &buf)

	// Log a message
	DefaultLogger.Error("Test message")

	// Check that message was logged
	if !strings.Contains(buf.String(), "Test message") {
		t.Errorf("Message not logged to buffer using global Error function")
	}

	// Restore original settings
	DefaultLogger.SetOutput(ErrorLevel, os.Stderr)
}

func TestLDAPProtocolLogger(t *testing.T) {
	// Create buffers to capture output
	var buf bytes.Buffer

	// Store original settings
	originalLevel := LDAPProtocolLogger.GetLevel()
	defer LDAPProtocolLogger.SetLevel(originalLevel)

	// Test the LDAP-specific logger
	LDAPProtocolLogger.SetLevel(DebugLevel)
	LDAPProtocolLogger.SetOutput(DebugLevel, &buf)

	// Log a message
	LDAPProtocolLogger.Debug("Test LDAP protocol message")

	// Check that message was logged
	if !strings.Contains(buf.String(), "Test LDAP protocol message") {
		t.Errorf("Message not logged to buffer using LDAP protocol logger")
	}

	// Check prefix contains "LDAP"
	if !strings.Contains(buf.String(), "[LDAP:") {
		t.Errorf("LDAP logger should have 'LDAP' in prefix, got: %s", buf.String())
	}

	// Restore original settings
	LDAPProtocolLogger.SetOutput(DebugLevel, os.Stdout)
}
