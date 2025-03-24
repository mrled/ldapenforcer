package config

import (
	"path/filepath"
	"testing"

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
