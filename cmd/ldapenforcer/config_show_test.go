package ldapenforcer

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/mrled/ldapenforcer/internal/config"
)

// TestConfigShowDefaults tests that running with no options set in the file/env/args produces defaults
func TestConfigShowDefaults(t *testing.T) {
	if os.Getenv("TEST_CONFIG_SHOW_INTEGRATION") != "true" {
		t.Skip("Skipping integration test. Set TEST_CONFIG_SHOW_INTEGRATION=true to run")
	}

	// Build the binary
	bin := buildBinary(t)
	defer os.Remove(bin)

	// Run the binary with only config-show
	cmd := exec.Command(bin, "config-show")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute command: %v, output: %s", err, out)
	}

	// Output the command result for debugging
	t.Logf("Command output: %s", out)

	// Extract only the TOML part from the output - ignore log lines
	tomlStart := strings.Index(string(out), "[ldapenforcer]")
	if tomlStart == -1 {
		t.Fatalf("Failed to find [ldapenforcer] section in output: %s", out)
	}
	tomlOutput := string(out)[tomlStart:]

	// Parse the TOML output
	var result map[string]interface{}
	if err := toml.Unmarshal([]byte(tomlOutput), &result); err != nil {
		t.Fatalf("Failed to parse TOML output: %v, output: %s", err, tomlOutput)
	}

	// Access the ldapenforcer section
	ldapenforcer, ok := result["ldapenforcer"].(map[string]interface{})
	if !ok {
		t.Fatalf("No ldapenforcer section found in output")
	}

	// Check default values
	expected := map[string]interface{}{
		"main_log_level": "INFO",
		"ldap_log_level": "INFO",
	}

	for key, expectedValue := range expected {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v but got %v", key, expectedValue, actualValue)
		}
	}
}

// TestConfigShowFileAndEnv tests that running with file/env has env override file
func TestConfigShowFileAndEnv(t *testing.T) {
	if os.Getenv("TEST_CONFIG_SHOW_INTEGRATION") != "true" {
		t.Skip("Skipping integration test. Set TEST_CONFIG_SHOW_INTEGRATION=true to run")
	}

	// Build the binary
	bin := buildBinary(t)
	defer os.Remove(bin)

	// Get the test config file path
	configPath, err := filepath.Abs("../../internal/config/testdata/config.toml")
	if err != nil {
		t.Fatalf("Failed to get absolute path for test config: %v", err)
	}

	// Save original environment variables
	origVars := map[string]string{
		"LDAPENFORCER_URI":                os.Getenv("LDAPENFORCER_URI"),
		"LDAPENFORCER_BIND_DN":            os.Getenv("LDAPENFORCER_BIND_DN"),
		"LDAPENFORCER_LOG_LEVEL":          os.Getenv("LDAPENFORCER_LOG_LEVEL"),
		"LDAPENFORCER_LDAP_LOG_LEVEL":     os.Getenv("LDAPENFORCER_LDAP_LOG_LEVEL"),
		"LDAPENFORCER_ENFORCED_PEOPLE_OU": os.Getenv("LDAPENFORCER_ENFORCED_PEOPLE_OU"),
	}

	// Restore original environment after test
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test environment variables that should override config file
	os.Setenv("LDAPENFORCER_URI", "ldap://envtest.com:389")
	os.Setenv("LDAPENFORCER_LOG_LEVEL", "DEBUG")

	// Verify environment variable was set
	t.Logf("LDAPENFORCER_LOG_LEVEL is set to: %s", os.Getenv("LDAPENFORCER_LOG_LEVEL"))

	// Run the binary with config file and add debug flags
	cmd := exec.Command(bin, "--config", configPath, "--log-level", "DEBUG", "config-show")
	cmd.Env = os.Environ() // Make sure env vars are passed to the command
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute command: %v, output: %s", err, out)
	}

	// Output the command result for debugging
	t.Logf("Command output: %s", out)

	// Extract only the TOML part from the output - ignore log lines
	tomlStart := strings.Index(string(out), "[ldapenforcer]")
	if tomlStart == -1 {
		t.Fatalf("Failed to find [ldapenforcer] section in output: %s", out)
	}
	tomlOutput := string(out)[tomlStart:]

	// Parse the TOML output
	var result map[string]interface{}
	if err := toml.Unmarshal([]byte(tomlOutput), &result); err != nil {
		t.Fatalf("Failed to parse TOML output: %v, output: %s", err, tomlOutput)
	}

	// Access the ldapenforcer section
	ldapenforcer, ok := result["ldapenforcer"].(map[string]interface{})
	if !ok {
		t.Fatalf("No ldapenforcer section found in output")
	}

	// Config file value tests
	expectedFromFile := map[string]interface{}{
		"bind_dn":             "cn=admin,dc=example,dc=com",
		"password":            "admin_password",
		"enforced_svcacct_ou": "ou=managed,ou=svcaccts,dc=example,dc=com",
		"enforced_group_ou":   "ou=managed,ou=groups,dc=example,dc=com",
	}

	for key, expectedValue := range expectedFromFile {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v from file but got %v", key, expectedValue, actualValue)
		}
	}

	// Environment variable override tests
	expectedFromEnv := map[string]interface{}{
		"uri":            "ldap://envtest.com:389",
		"main_log_level": "DEBUG",
	}

	for key, expectedValue := range expectedFromEnv {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v from env but got %v", key, expectedValue, actualValue)
		}
	}

	// Check that enforced_people_ou comes from includes (overrides main config)
	peopleOu, exists := ldapenforcer["enforced_people_ou"]
	if !exists {
		t.Errorf("Expected key enforced_people_ou not found in output")
	} else if peopleOu != "ou=managed-override,ou=people,dc=example,dc=com" {
		t.Errorf("For enforced_people_ou, expected value from included file but got %v", peopleOu)
	}
}

// TestConfigShowEnvAndArgs tests that running with env/args has args override env
func TestConfigShowEnvAndArgs(t *testing.T) {
	if os.Getenv("TEST_CONFIG_SHOW_INTEGRATION") != "true" {
		t.Skip("Skipping integration test. Set TEST_CONFIG_SHOW_INTEGRATION=true to run")
	}

	// Build the binary
	bin := buildBinary(t)
	defer os.Remove(bin)

	// Save original environment variables
	origVars := map[string]string{
		"LDAPENFORCER_URI":                os.Getenv("LDAPENFORCER_URI"),
		"LDAPENFORCER_BIND_DN":            os.Getenv("LDAPENFORCER_BIND_DN"),
		"LDAPENFORCER_LOG_LEVEL":          os.Getenv("LDAPENFORCER_LOG_LEVEL"),
		"LDAPENFORCER_LDAP_LOG_LEVEL":     os.Getenv("LDAPENFORCER_LDAP_LOG_LEVEL"),
		"LDAPENFORCER_ENFORCED_PEOPLE_OU": os.Getenv("LDAPENFORCER_ENFORCED_PEOPLE_OU"),
	}

	// Restore original environment after test
	defer func() {
		for k, v := range origVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test environment variables
	os.Setenv("LDAPENFORCER_URI", "ldap://envtest.com:389")
	os.Setenv("LDAPENFORCER_BIND_DN", "cn=envuser,dc=example,dc=com")
	os.Setenv("LDAPENFORCER_LOG_LEVEL", "DEBUG")
	os.Setenv("LDAPENFORCER_LDAP_LOG_LEVEL", "INFO")
	os.Setenv("LDAPENFORCER_ENFORCED_PEOPLE_OU", "ou=env-managed,ou=people,dc=example,dc=com")

	// Run the binary with command line arguments that override environment variables
	cmd := exec.Command(bin,
		"--ldap-uri", "ldap://argtest.com:389",
		"--log-level", "TRACE",
		"config-show")
	cmd.Env = os.Environ() // Make sure env vars are passed to the command
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to execute command: %v, output: %s", err, out)
	}

	// Output the command result for debugging
	t.Logf("Command output: %s", out)

	// Extract only the TOML part from the output - ignore log lines
	tomlStart := strings.Index(string(out), "[ldapenforcer]")
	if tomlStart == -1 {
		t.Fatalf("Failed to find [ldapenforcer] section in output: %s", out)
	}
	tomlOutput := string(out)[tomlStart:]

	// Parse the TOML output
	var result map[string]interface{}
	if err := toml.Unmarshal([]byte(tomlOutput), &result); err != nil {
		t.Fatalf("Failed to parse TOML output: %v, output: %s", err, tomlOutput)
	}

	// Access the ldapenforcer section
	ldapenforcer, ok := result["ldapenforcer"].(map[string]interface{})
	if !ok {
		t.Fatalf("No ldapenforcer section found in output")
	}

	// Environment variable tests
	expectedFromEnv := map[string]interface{}{
		"bind_dn":            "cn=envuser,dc=example,dc=com",
		"ldap_log_level":     "INFO",
		"enforced_people_ou": "ou=env-managed,ou=people,dc=example,dc=com",
	}

	for key, expectedValue := range expectedFromEnv {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v from env but got %v", key, expectedValue, actualValue)
		}
	}

	// Command line argument override tests
	expectedFromArgs := map[string]interface{}{
		"uri":            "ldap://argtest.com:389",
		"main_log_level": "TRACE",
	}

	for key, expectedValue := range expectedFromArgs {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v from args but got %v", key, expectedValue, actualValue)
		}
	}
}

// Helper function to build the binary for testing
func buildBinary(t *testing.T) string {
	t.Helper()

	// Create a temporary file for the binary
	tmpFile, err := os.CreateTemp("", "ldapenforcer-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()

	// Build the binary
	cmd := exec.Command("go", "build", "-o", tmpFile.Name(), "../../")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to build binary: %v, output: %s", err, output)
	}

	return tmpFile.Name()
}

// TestEncodeTOML tests that encoding a config to TOML works properly
// This is a unit test that directly tests the encoding logic
// without invoking the full CLI command
func TestEncodeTOML(t *testing.T) {
	// Create a test config
	cfg := &config.Config{}

	// Manually set defaults
	cfg.LDAPEnforcer.MainLogLevel = "DEBUG"
	cfg.LDAPEnforcer.LDAPLogLevel = "INFO"
	cfg.LDAPEnforcer.URI = "ldap://testserver.com:389"
	cfg.LDAPEnforcer.BindDN = "cn=admin,dc=test,dc=com"
	cfg.LDAPEnforcer.EnforcedPeopleOU = "ou=people,dc=test,dc=com"
	cfg.LDAPEnforcer.EnforcedSvcAcctOU = "ou=svcaccts,dc=test,dc=com"
	cfg.LDAPEnforcer.EnforcedGroupOU = "ou=groups,dc=test,dc=com"

	// Use the same encoding logic as the config-show command
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(cfg)
	if err != nil {
		t.Fatalf("Error encoding configuration: %v", err)
	}

	// Read the encoded TOML
	output := buf.String()

	// Parse the TOML output
	var result map[string]interface{}
	if err := toml.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse TOML output: %v, output: %s", err, output)
	}

	// Access the ldapenforcer section
	ldapenforcer, ok := result["ldapenforcer"].(map[string]interface{})
	if !ok {
		t.Fatalf("No ldapenforcer section found in output")
	}

	// Check values
	expected := map[string]interface{}{
		"uri":                 "ldap://testserver.com:389",
		"bind_dn":             "cn=admin,dc=test,dc=com",
		"main_log_level":      "DEBUG",
		"ldap_log_level":      "INFO",
		"enforced_people_ou":  "ou=people,dc=test,dc=com",
		"enforced_svcacct_ou": "ou=svcaccts,dc=test,dc=com",
		"enforced_group_ou":   "ou=groups,dc=test,dc=com",
	}

	for key, expectedValue := range expected {
		actualValue, exists := ldapenforcer[key]
		if !exists {
			t.Errorf("Expected key %s not found in output", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("For key %s, expected %v but got %v", key, expectedValue, actualValue)
		}
	}
}

// Helper function to execute a command and capture its output
func execCommand(t *testing.T, binary string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to execute command %s %s: %v\nStdout: %s\nStderr: %s",
			binary, strings.Join(args, " "), err, stdout.String(), stderr.String())
	}

	return stdout.String()
}
