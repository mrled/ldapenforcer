package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mrled/ldapenforcer/internal/model"
	"github.com/spf13/pflag"
)

func TestLoadConfig(t *testing.T) {
	// Get the absolute path to the test config file
	configPath, err := filepath.Abs("testdata/config.toml")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Load the config
	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check values
	if config.LDAPEnforcer.URI != "ldap://example.com:389" {
		t.Errorf("Expected URI 'ldap://example.com:389', got '%s'", config.LDAPEnforcer.URI)
	}
	if config.LDAPEnforcer.BindDN != "cn=admin,dc=example,dc=com" {
		t.Errorf("Expected BindDN 'cn=admin,dc=example,dc=com', got '%s'", config.LDAPEnforcer.BindDN)
	}
	if config.LDAPEnforcer.Password != "admin_password" {
		t.Errorf("Expected Password 'admin_password', got '%s'", config.LDAPEnforcer.Password)
	}
	if config.LDAPEnforcer.Logging.Level != "INFO" {
		t.Errorf("Expected Logging.Level 'INFO', got '%s'", config.LDAPEnforcer.Logging.Level)
	}
	if config.LDAPEnforcer.PeopleBaseDN != "ou=people,dc=example,dc=com" {
		t.Errorf("Expected PeopleBaseDN 'ou=people,dc=example,dc=com', got '%s'", config.LDAPEnforcer.PeopleBaseDN)
	}
	if config.LDAPEnforcer.SvcAcctBaseDN != "ou=svcaccts,dc=example,dc=com" {
		t.Errorf("Expected SvcAcctBaseDN 'ou=svcaccts,dc=example,dc=com', got '%s'", config.LDAPEnforcer.SvcAcctBaseDN)
	}
	if config.LDAPEnforcer.GroupBaseDN != "ou=groups,dc=example,dc=com" {
		t.Errorf("Expected GroupBaseDN 'ou=groups,dc=example,dc=com', got '%s'", config.LDAPEnforcer.GroupBaseDN)
	}

	// Check that includes were processed
	if config.LDAPEnforcer.ManagedOU != "managed-override" {
		t.Errorf("Expected ManagedOU 'managed-override' from included file, got '%s'", config.LDAPEnforcer.ManagedOU)
	}
}

func TestGetPassword(t *testing.T) {
	// Set the configDir for testing
	absPath, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatalf("Failed to get absolute path for testdata: %v", err)
	}
	configDir = absPath

	tests := []struct {
		name                    string
		password                string
		passwordFile            string
		passwordCommand         string
		passwordCommandViaShell bool
		expectedResult          string
		expectError             bool
	}{
		{
			name:                    "Direct password",
			password:                "direct_password",
			passwordFile:            "",
			passwordCommand:         "",
			passwordCommandViaShell: false,
			expectedResult:          "direct_password",
			expectError:             false,
		},
		{
			name:                    "Password from file",
			password:                "",
			passwordFile:            "password.txt",
			passwordCommand:         "",
			passwordCommandViaShell: false,
			expectedResult:          "secret_password",
			expectError:             false,
		},
		{
			name:                    "Password with whitespace",
			password:                "",
			passwordFile:            "password_with_whitespace.txt",
			passwordCommand:         "",
			passwordCommandViaShell: false,
			expectedResult:          "secret_password_with_whitespace",
			expectError:             false,
		},
		{
			name:                    "Nonexistent file",
			password:                "",
			passwordFile:            "nonexistent.txt",
			passwordCommand:         "",
			passwordCommandViaShell: false,
			expectedResult:          "",
			expectError:             true,
		},
		{
			name:                    "Password from command",
			password:                "",
			passwordFile:            "",
			passwordCommand:         "echo command_password",
			passwordCommandViaShell: false,
			expectedResult:          "command_password",
			expectError:             false,
		},
		{
			name:                    "Password from command via shell",
			password:                "",
			passwordFile:            "",
			passwordCommand:         "echo command_password_shell",
			passwordCommandViaShell: true,
			expectedResult:          "command_password_shell",
			expectError:             false,
		},
		{
			name:                    "Failing command",
			password:                "",
			passwordFile:            "",
			passwordCommand:         "false",
			passwordCommandViaShell: false,
			expectedResult:          "",
			expectError:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			config.LDAPEnforcer.Password = tt.password
			config.LDAPEnforcer.PasswordFile = tt.passwordFile
			config.LDAPEnforcer.PasswordCommand = tt.passwordCommand
			config.LDAPEnforcer.PasswordCommandViaShell = tt.passwordCommandViaShell

			result, err := config.GetPassword()

			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("Expected password '%s', got '%s'", tt.expectedResult, result)
			}
		})
	}
}

func TestMergeWithFlags(t *testing.T) {
	// Create a test config
	config := &Config{}

	// Create test flags
	flags := NewTestFlagSet()
	AddFlags(flags)

	// Set some flag values
	flags.Set("ldap-uri", "ldap://flagtest.com")
	flags.Set("bind-dn", "cn=flaguser,dc=example,dc=com")
	flags.Set("password-command", "echo password")
	flags.Set("password-command-via-shell", "true")
	flags.Set("ca-cert-file", "/path/to/ca.crt")
	flags.Set("people-base-dn", "ou=people,dc=flagtest,dc=com")
	flags.Set("svcacct-base-dn", "ou=svcaccts,dc=flagtest,dc=com")
	flags.Set("managed-ou", "flag-managed")

	// Merge with config
	config.MergeWithFlags(flags)

	// Check values
	if config.LDAPEnforcer.URI != "ldap://flagtest.com" {
		t.Errorf("Expected URI 'ldap://flagtest.com', got '%s'", config.LDAPEnforcer.URI)
	}
	if config.LDAPEnforcer.BindDN != "cn=flaguser,dc=example,dc=com" {
		t.Errorf("Expected BindDN 'cn=flaguser,dc=example,dc=com', got '%s'", config.LDAPEnforcer.BindDN)
	}
	if config.LDAPEnforcer.PasswordCommand != "echo password" {
		t.Errorf("Expected PasswordCommand 'echo password', got '%s'", config.LDAPEnforcer.PasswordCommand)
	}
	if !config.LDAPEnforcer.PasswordCommandViaShell {
		t.Errorf("Expected PasswordCommandViaShell 'true', got '%v'", config.LDAPEnforcer.PasswordCommandViaShell)
	}
	if config.LDAPEnforcer.CACertFile != "/path/to/ca.crt" {
		t.Errorf("Expected CACertFile '/path/to/ca.crt', got '%s'", config.LDAPEnforcer.CACertFile)
	}
	if config.LDAPEnforcer.PeopleBaseDN != "ou=people,dc=flagtest,dc=com" {
		t.Errorf("Expected PeopleBaseDN 'ou=people,dc=flagtest,dc=com', got '%s'", config.LDAPEnforcer.PeopleBaseDN)
	}
	if config.LDAPEnforcer.SvcAcctBaseDN != "ou=svcaccts,dc=flagtest,dc=com" {
		t.Errorf("Expected SvcAcctBaseDN 'ou=svcaccts,dc=flagtest,dc=com', got '%s'", config.LDAPEnforcer.SvcAcctBaseDN)
	}
	if config.LDAPEnforcer.ManagedOU != "flag-managed" {
		t.Errorf("Expected ManagedOU 'flag-managed', got '%s'", config.LDAPEnforcer.ManagedOU)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					URI:           "ldap://example.com",
					BindDN:        "cn=admin,dc=example,dc=com",
					Password:      "password",
					PeopleBaseDN:  "ou=people,dc=example,dc=com",
					SvcAcctBaseDN: "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:   "ou=groups,dc=example,dc=com",
					ManagedOU:     "managed",
				},
			},
			expectError: false,
		},
		{
			name: "Missing URI",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					BindDN:        "cn=admin,dc=example,dc=com",
					Password:      "password",
					PeopleBaseDN:  "ou=people,dc=example,dc=com",
					SvcAcctBaseDN: "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:   "ou=groups,dc=example,dc=com",
					ManagedOU:     "managed",
				},
			},
			expectError: true,
		},
		{
			name: "Missing password, password file, and password command",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					URI:           "ldap://example.com",
					BindDN:        "cn=admin,dc=example,dc=com",
					PeopleBaseDN:  "ou=people,dc=example,dc=com",
					SvcAcctBaseDN: "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:   "ou=groups,dc=example,dc=com",
					ManagedOU:     "managed",
				},
			},
			expectError: true,
		},
		{
			name: "With password file instead of password",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					URI:           "ldap://example.com",
					BindDN:        "cn=admin,dc=example,dc=com",
					PasswordFile:  "path/to/password.txt",
					PeopleBaseDN:  "ou=people,dc=example,dc=com",
					SvcAcctBaseDN: "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:   "ou=groups,dc=example,dc=com",
					ManagedOU:     "managed",
				},
			},
			expectError: false,
		},
		{
			name: "With password command instead of password",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					URI:             "ldap://example.com",
					BindDN:          "cn=admin,dc=example,dc=com",
					PasswordCommand: "echo password",
					PeopleBaseDN:    "ou=people,dc=example,dc=com",
					SvcAcctBaseDN:   "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:     "ou=groups,dc=example,dc=com",
					ManagedOU:       "managed",
				},
			},
			expectError: false,
		},
		{
			name: "With password command via shell",
			config: &Config{
				LDAPEnforcer: LDAPEnforcerConfig{
					URI:                     "ldap://example.com",
					BindDN:                  "cn=admin,dc=example,dc=com",
					PasswordCommand:         "echo password",
					PasswordCommandViaShell: true,
					PeopleBaseDN:            "ou=people,dc=example,dc=com",
					SvcAcctBaseDN:           "ou=svcaccts,dc=example,dc=com",
					GroupBaseDN:             "ou=groups,dc=example,dc=com",
					ManagedOU:               "managed",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}
		})
	}
}

func TestParseCommandString(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		expected    []string
		expectError bool
	}{
		{
			name:        "Simple command",
			command:     "echo hello",
			expected:    []string{"echo", "hello"},
			expectError: false,
		},
		{
			name:        "Command with quotes",
			command:     "echo \"hello world\"",
			expected:    []string{"echo", "hello world"},
			expectError: false,
		},
		{
			name:        "Command with single quotes",
			command:     "echo 'hello world'",
			expected:    []string{"echo", "hello world"},
			expectError: false,
		},
		{
			name:        "Command with nested quotes",
			command:     "echo \"hello 'world'\"",
			expected:    []string{"echo", "hello 'world'"},
			expectError: false,
		},
		{
			name:        "Command with multiple options",
			command:     "pass show --clip ldap/admin",
			expected:    []string{"pass", "show", "--clip", "ldap/admin"},
			expectError: false,
		},
		{
			name:        "Unclosed quotes",
			command:     "echo \"hello world",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCommandString(tt.command)

			if tt.expectError && err == nil {
				t.Errorf("Expected an error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}

			if !tt.expectError {
				if len(result) != len(tt.expected) {
					t.Errorf("Expected %d parts, got %d", len(tt.expected), len(result))
				} else {
					for i, part := range result {
						if part != tt.expected[i] {
							t.Errorf("Part %d: expected '%s', got '%s'", i, tt.expected[i], part)
						}
					}
				}
			}
		})
	}
}

// Helper function to create a test flag set
func NewTestFlagSet() *pflag.FlagSet {
	return pflag.NewFlagSet("test", pflag.ContinueOnError)
}

func TestMergeWithEnv(t *testing.T) {
	// Save original environment variables
	origVars := map[string]string{
		"LDAPENFORCER_URI":                        os.Getenv("LDAPENFORCER_URI"),
		"LDAPENFORCER_BIND_DN":                    os.Getenv("LDAPENFORCER_BIND_DN"),
		"LDAPENFORCER_PASSWORD":                   os.Getenv("LDAPENFORCER_PASSWORD"),
		"LDAPENFORCER_PASSWORD_FILE":              os.Getenv("LDAPENFORCER_PASSWORD_FILE"),
		"LDAPENFORCER_PASSWORD_COMMAND":           os.Getenv("LDAPENFORCER_PASSWORD_COMMAND"),
		"LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL": os.Getenv("LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL"),
		"LDAPENFORCER_CA_CERT_FILE":               os.Getenv("LDAPENFORCER_CA_CERT_FILE"),
		"LDAPENFORCER_LOG_LEVEL":                  os.Getenv("LDAPENFORCER_LOG_LEVEL"),
		"LDAPENFORCER_LDAP_LOG_LEVEL":             os.Getenv("LDAPENFORCER_LDAP_LOG_LEVEL"),
		"LDAPENFORCER_PEOPLE_BASE_DN":             os.Getenv("LDAPENFORCER_PEOPLE_BASE_DN"),
		"LDAPENFORCER_SVCACCT_BASE_DN":            os.Getenv("LDAPENFORCER_SVCACCT_BASE_DN"),
		"LDAPENFORCER_GROUP_BASE_DN":              os.Getenv("LDAPENFORCER_GROUP_BASE_DN"),
		"LDAPENFORCER_MANAGED_OU":                 os.Getenv("LDAPENFORCER_MANAGED_OU"),
		"LDAPENFORCER_INCLUDES":                   os.Getenv("LDAPENFORCER_INCLUDES"),
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
	os.Setenv("LDAPENFORCER_URI", "ldap://envtest.com")
	os.Setenv("LDAPENFORCER_BIND_DN", "cn=envuser,dc=example,dc=com")
	os.Setenv("LDAPENFORCER_PASSWORD", "env_password")
	os.Setenv("LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL", "true")
	os.Setenv("LDAPENFORCER_CA_CERT_FILE", "/path/from/env/ca.crt")
	os.Setenv("LDAPENFORCER_LOG_LEVEL", "INFO")
	os.Setenv("LDAPENFORCER_LDAP_LOG_LEVEL", "DEBUG")
	os.Setenv("LDAPENFORCER_PEOPLE_BASE_DN", "ou=people,dc=envtest,dc=com")
	os.Setenv("LDAPENFORCER_SVCACCT_BASE_DN", "ou=svcaccts,dc=envtest,dc=com")
	os.Setenv("LDAPENFORCER_GROUP_BASE_DN", "ou=groups,dc=envtest,dc=com")
	os.Setenv("LDAPENFORCER_MANAGED_OU", "env-managed")
	os.Setenv("LDAPENFORCER_INCLUDES", "env1.toml, env2.toml")

	// Create a test config
	config := &Config{}

	// Initialize the config structure to avoid nil pointers
	config.LDAPEnforcer.Person = make(map[string]*model.Person)
	config.LDAPEnforcer.SvcAcct = make(map[string]*model.SvcAcct)
	config.LDAPEnforcer.Group = make(map[string]*model.Group)
	config.LDAPEnforcer.Includes = make([]string, 0)

	// Load from environment
	config.MergeWithEnv()

	// Check values
	if config.LDAPEnforcer.URI != "ldap://envtest.com" {
		t.Errorf("Expected URI 'ldap://envtest.com', got '%s'", config.LDAPEnforcer.URI)
	}
	if config.LDAPEnforcer.BindDN != "cn=envuser,dc=example,dc=com" {
		t.Errorf("Expected BindDN 'cn=envuser,dc=example,dc=com', got '%s'", config.LDAPEnforcer.BindDN)
	}
	if config.LDAPEnforcer.Password != "env_password" {
		t.Errorf("Expected Password 'env_password', got '%s'", config.LDAPEnforcer.Password)
	}
	if !config.LDAPEnforcer.PasswordCommandViaShell {
		t.Errorf("Expected PasswordCommandViaShell 'true', got '%v'", config.LDAPEnforcer.PasswordCommandViaShell)
	}
	if config.LDAPEnforcer.CACertFile != "/path/from/env/ca.crt" {
		t.Errorf("Expected CACertFile '/path/from/env/ca.crt', got '%s'", config.LDAPEnforcer.CACertFile)
	}
	if config.LDAPEnforcer.Logging.Level != "INFO" {
		t.Errorf("Expected Logging.Level 'INFO', got '%s'", config.LDAPEnforcer.Logging.Level)
	}
	if config.LDAPEnforcer.Logging.LDAP.Level != "DEBUG" {
		t.Errorf("Expected Logging.LDAP.Level 'DEBUG', got '%s'", config.LDAPEnforcer.Logging.LDAP.Level)
	}
	if config.LDAPEnforcer.PeopleBaseDN != "ou=people,dc=envtest,dc=com" {
		t.Errorf("Expected PeopleBaseDN 'ou=people,dc=envtest,dc=com', got '%s'", config.LDAPEnforcer.PeopleBaseDN)
	}
	if config.LDAPEnforcer.SvcAcctBaseDN != "ou=svcaccts,dc=envtest,dc=com" {
		t.Errorf("Expected SvcAcctBaseDN 'ou=svcaccts,dc=envtest,dc=com', got '%s'", config.LDAPEnforcer.SvcAcctBaseDN)
	}
	if config.LDAPEnforcer.GroupBaseDN != "ou=groups,dc=envtest,dc=com" {
		t.Errorf("Expected GroupBaseDN 'ou=groups,dc=envtest,dc=com', got '%s'", config.LDAPEnforcer.GroupBaseDN)
	}
	if config.LDAPEnforcer.ManagedOU != "env-managed" {
		t.Errorf("Expected ManagedOU 'env-managed', got '%s'", config.LDAPEnforcer.ManagedOU)
	}

	// Check includes (should have been cleaned up from comma-separated string)
	expectedIncludes := []string{"env1.toml", "env2.toml"}
	if len(config.LDAPEnforcer.Includes) != len(expectedIncludes) {
		t.Errorf("Expected %d includes, got %d", len(expectedIncludes), len(config.LDAPEnforcer.Includes))
	} else {
		for i, include := range config.LDAPEnforcer.Includes {
			if include != expectedIncludes[i] {
				t.Errorf("Expected include '%s', got '%s'", expectedIncludes[i], include)
			}
		}
	}
}
