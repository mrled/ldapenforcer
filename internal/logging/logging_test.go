package logging

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	// Store initial level to restore after test
	initialLevel := currentLevel
	defer SetLevel(initialLevel)

	// Test setting and getting level
	levels := []LogLevel{
		ErrorLevel,
		WarnLevel,
		InfoLevel,
		DebugLevel,
		LDAPLevel,
		TraceLevel,
	}

	for _, level := range levels {
		SetLevel(level)
		if GetLevel() != level {
			t.Errorf("Expected level %v, got %v", level, GetLevel())
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
		{LDAPLevel, "LDAP"},
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
		{"LDAP", LDAPLevel, false},
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

func TestLoggingOutputs(t *testing.T) {
	// Store original levels and outputs
	originalLevel := currentLevel
	
	// Create buffers to capture output
	var errorBuf, warnBuf, infoBuf, debugBuf, ldapBuf, traceBuf bytes.Buffer
	
	// Redirect logger outputs
	errorLogger.SetOutput(&errorBuf)
	warnLogger.SetOutput(&warnBuf)
	infoLogger.SetOutput(&infoBuf)
	debugLogger.SetOutput(&debugBuf)
	ldapLogger.SetOutput(&ldapBuf)
	traceLogger.SetOutput(&traceBuf)
	
	// Test messages
	errorMsg := "Error message"
	warnMsg := "Warning message"
	infoMsg := "Info message"
	debugMsg := "Debug message"
	ldapMsg := "LDAP message"
	traceMsg := "Trace message"
	
	// Test at ERROR level
	SetLevel(ErrorLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("ERROR level should log error messages")
	}
	if warnBuf.Len() > 0 || infoBuf.Len() > 0 || debugBuf.Len() > 0 || ldapBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("ERROR level should not log warn, info, debug, ldap, or trace messages")
	}
	
	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	ldapBuf.Reset()
	traceBuf.Reset()
	
	// Test at WARN level
	SetLevel(WarnLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("WARN level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("WARN level should log warning messages")
	}
	if infoBuf.Len() > 0 || debugBuf.Len() > 0 || ldapBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("WARN level should not log info, debug, ldap, or trace messages")
	}
	
	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	ldapBuf.Reset()
	traceBuf.Reset()
	
	// Test at INFO level
	SetLevel(InfoLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("INFO level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("INFO level should log warning messages")
	}
	if !strings.Contains(infoBuf.String(), infoMsg) {
		t.Errorf("INFO level should log info messages")
	}
	if debugBuf.Len() > 0 || ldapBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("INFO level should not log debug, ldap, or trace messages")
	}
	
	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	ldapBuf.Reset()
	traceBuf.Reset()
	
	// Test at DEBUG level
	SetLevel(DebugLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
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
	if ldapBuf.Len() > 0 || traceBuf.Len() > 0 {
		t.Errorf("DEBUG level should not log ldap or trace messages")
	}
	
	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	ldapBuf.Reset()
	traceBuf.Reset()
	
	// Test at LDAP level
	SetLevel(LDAPLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
	if !strings.Contains(errorBuf.String(), errorMsg) {
		t.Errorf("LDAP level should log error messages")
	}
	if !strings.Contains(warnBuf.String(), warnMsg) {
		t.Errorf("LDAP level should log warning messages")
	}
	if !strings.Contains(infoBuf.String(), infoMsg) {
		t.Errorf("LDAP level should log info messages")
	}
	if !strings.Contains(debugBuf.String(), debugMsg) {
		t.Errorf("LDAP level should log debug messages")
	}
	if !strings.Contains(ldapBuf.String(), ldapMsg) {
		t.Errorf("LDAP level should log ldap messages")
	}
	if traceBuf.Len() > 0 {
		t.Errorf("LDAP level should not log trace messages")
	}
	
	// Reset buffers
	errorBuf.Reset()
	warnBuf.Reset()
	infoBuf.Reset()
	debugBuf.Reset()
	ldapBuf.Reset()
	traceBuf.Reset()
	
	// Test at TRACE level
	SetLevel(TraceLevel)
	Error(errorMsg)
	Warn(warnMsg)
	Info(infoMsg)
	Debug(debugMsg)
	LDAP(ldapMsg)
	Trace(traceMsg)
	
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
	if !strings.Contains(ldapBuf.String(), ldapMsg) {
		t.Errorf("TRACE level should log ldap messages")
	}
	if !strings.Contains(traceBuf.String(), traceMsg) {
		t.Errorf("TRACE level should log trace messages")
	}
	
	// Restore original level
	SetLevel(originalLevel)
}

func TestSetOutput(t *testing.T) {
	// Create buffer and test redirection
	var buf bytes.Buffer
	
	// Test setting output for each level
	levels := []LogLevel{
		ErrorLevel,
		WarnLevel,
		InfoLevel,
		DebugLevel,
		LDAPLevel,
		TraceLevel,
	}
	
	messages := map[LogLevel]string{
		ErrorLevel: "Error test",
		WarnLevel:  "Warn test",
		InfoLevel:  "Info test",
		DebugLevel: "Debug test",
		LDAPLevel:  "LDAP test",
		TraceLevel: "Trace test",
	}
	
	// Store original level
	originalLevel := currentLevel
	SetLevel(TraceLevel) // Set to highest level so all messages are logged
	
	for _, level := range levels {
		// Reset buffer
		buf.Reset()
		
		// Set output for this level only
		SetOutput(level, &buf)
		
		// Log a message at this level
		switch level {
		case ErrorLevel:
			Error(messages[level])
		case WarnLevel:
			Warn(messages[level])
		case InfoLevel:
			Info(messages[level])
		case DebugLevel:
			Debug(messages[level])
		case LDAPLevel:
			LDAP(messages[level])
		case TraceLevel:
			Trace(messages[level])
		}
		
		// Check that message was logged to buffer
		if !strings.Contains(buf.String(), messages[level]) {
			t.Errorf("Message for level %s not logged to buffer", GetLevelName(level))
		}
		
		// Reset output to original
		switch level {
		case ErrorLevel:
			errorLogger.SetOutput(os.Stderr)
		case WarnLevel:
			warnLogger.SetOutput(os.Stderr)
		case InfoLevel:
			infoLogger.SetOutput(os.Stdout)
		case DebugLevel:
			debugLogger.SetOutput(os.Stdout)
		case LDAPLevel:
			ldapLogger.SetOutput(os.Stdout)
		case TraceLevel:
			traceLogger.SetOutput(os.Stdout)
		}
	}
	
	// Restore original level
	SetLevel(originalLevel)
}

func TestSetPrefix(t *testing.T) {
	// Create buffer to capture output
	var buf bytes.Buffer
	
	// Store original level and prefix
	originalLevel := currentLevel
	originalPrefix := errorLogger.Prefix()
	
	// Set level to error for testing
	SetLevel(ErrorLevel)
	
	// Set output to buffer
	errorLogger.SetOutput(&buf)
	
	// Test setting prefix
	testPrefix := "[TEST] "
	SetPrefix(ErrorLevel, testPrefix)
	
	if errorLogger.Prefix() != testPrefix {
		t.Errorf("Expected prefix '%s', got '%s'", testPrefix, errorLogger.Prefix())
	}
	
	// Log a message
	Error("Test message")
	
	// Check that message was logged with new prefix
	if !strings.Contains(buf.String(), testPrefix) {
		t.Errorf("Message not logged with expected prefix '%s'", testPrefix)
	}
	
	// Restore original settings
	SetLevel(originalLevel)
	SetPrefix(ErrorLevel, originalPrefix)
	errorLogger.SetOutput(os.Stderr)
}

func TestSetFlags(t *testing.T) {
	// Create buffer to capture output
	var buf bytes.Buffer
	
	// Store original settings
	originalLevel := currentLevel
	originalFlags := errorLogger.Flags()
	
	// Set level to error for testing
	SetLevel(ErrorLevel)
	
	// Set output to buffer
	errorLogger.SetOutput(&buf)
	
	// Test setting flags
	testFlags := 0 // No flags
	SetFlags(ErrorLevel, testFlags)
	
	if errorLogger.Flags() != testFlags {
		t.Errorf("Expected flags %d, got %d", testFlags, errorLogger.Flags())
	}
	
	// Log a message
	Error("Test message")
	
	// Check that message was logged with no date/time
	if strings.Contains(buf.String(), ":") { // Date/time usually contains colons
		t.Errorf("Message logged with unexpected format (should have no date/time with flags=0)")
	}
	
	// Restore original settings
	SetLevel(originalLevel)
	SetFlags(ErrorLevel, originalFlags)
	errorLogger.SetOutput(os.Stderr)
}