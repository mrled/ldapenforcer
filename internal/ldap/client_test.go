package ldap

import (
	"testing"

	"github.com/mrled/ldapenforcer/internal/config"
)

func TestGetOUFromDN(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "Single-level OU",
			dn:       "ou=managed,dc=example,dc=com",
			expected: "managed",
		},
		{
			name:     "Nested OU",
			dn:       "ou=people,ou=managed,dc=example,dc=com",
			expected: "people",
		},
		{
			name:     "Complex DN",
			dn:       "uid=john,ou=people,ou=managed,dc=example,dc=com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOUFromDN(tt.dn)
			if result != tt.expected {
				t.Errorf("getOUFromDN(%q) = %q, want %q", tt.dn, result, tt.expected)
			}
		})
	}
}

func TestDNCreation(t *testing.T) {
	testConfig := &config.Config{
		LDAPEnforcer: config.LDAPEnforcerConfig{
			EnforcedPeopleOU:  "ou=managed,ou=people,dc=example,dc=com",
			EnforcedSvcAcctOU: "ou=managed,ou=svcaccts,dc=example,dc=com",
			EnforcedGroupOU:   "ou=managed,ou=groups,dc=example,dc=com",
		},
	}

	client := &Client{
		config: testConfig,
	}

	tests := []struct {
		name     string
		fn       func(string) string
		input    string
		expected string
	}{
		{
			name:     "PersonToDN",
			fn:       client.PersonToDN,
			input:    "john",
			expected: "uid=john,ou=managed,ou=people,dc=example,dc=com",
		},
		{
			name:     "SvcAcctToDN",
			fn:       client.SvcAcctToDN,
			input:    "backup",
			expected: "uid=backup,ou=managed,ou=svcaccts,dc=example,dc=com",
		},
		{
			name:     "GroupToDN",
			fn:       client.GroupToDN,
			input:    "admins",
			expected: "cn=admins,ou=managed,ou=groups,dc=example,dc=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			if result != tt.expected {
				t.Errorf("%s(%q) = %q, want %q", tt.name, tt.input, result, tt.expected)
			}
		})
	}
}
