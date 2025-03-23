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
	tests := []struct {
		name           string
		password       string
		passwordFile   string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Direct password",
			password:       "direct_password",
			passwordFile:   "",
			expectedResult: "direct_password",
			expectError:    false,
		},
		{
			name:           "Password from file",
			password:       "",
			passwordFile:   "testdata/password.txt",
			expectedResult: "secret_password",
			expectError:    false,
		},
		{
			name:           "Password with whitespace",
			password:       "",
			passwordFile:   "testdata/password_with_whitespace.txt",
			expectedResult: "secret_password_with_whitespace",
			expectError:    false,
		},
		{
			name:           "Nonexistent file",
			password:       "",
			passwordFile:   "testdata/nonexistent.txt",
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			config.LDAPEnforcer.Password = tt.password
			config.LDAPEnforcer.PasswordFile = tt.passwordFile

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
			name: "Missing password and password file",
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

// Helper function to create a test flag set
func NewTestFlagSet() *pflag.FlagSet {
	return pflag.NewFlagSet("test", pflag.ContinueOnError)
}
