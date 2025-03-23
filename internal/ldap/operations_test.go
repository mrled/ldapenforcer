package ldap

import (
	"reflect"
	"testing"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/model"
)

func TestGetPersonAttributes(t *testing.T) {
	// Test a minimal person
	minimalPerson := &model.Person{
		Username: "jdoe",
		CN:       "John Doe",
	}
	minAttrs := GetPersonAttributes(minimalPerson)

	if minAttrs["cn"] == nil || minAttrs["cn"][0] != "John Doe" {
		t.Errorf("Expected CN attribute to be 'John Doe', got %v", minAttrs["cn"])
	}

	if minAttrs["sn"] == nil || minAttrs["sn"][0] != "Doe" {
		t.Errorf("Expected SN attribute to be 'Doe', got %v", minAttrs["sn"])
	}

	// Check object classes for non-POSIX person
	expectedClasses := []string{"top", "inetOrgPerson", "account"}
	if !reflect.DeepEqual(minAttrs["objectClass"], expectedClasses) {
		t.Errorf("Expected objectClass to be %v, got %v", expectedClasses, minAttrs["objectClass"])
	}

	// Test a complete person
	fullPerson := &model.Person{
		Username:  "jsmith",
		CN:        "Jane Smith",
		GivenName: "Jane",
		SN:        "Smith",
		Mail:      "jane.smith@example.com",
		Posix:     []int{1001, 1001},
	}
	fullAttrs := GetPersonAttributes(fullPerson)

	if fullAttrs["cn"] == nil || fullAttrs["cn"][0] != "Jane Smith" {
		t.Errorf("Expected CN attribute to be 'Jane Smith', got %v", fullAttrs["cn"])
	}

	if fullAttrs["givenName"] == nil || fullAttrs["givenName"][0] != "Jane" {
		t.Errorf("Expected givenName attribute to be 'Jane', got %v", fullAttrs["givenName"])
	}

	if fullAttrs["sn"] == nil || fullAttrs["sn"][0] != "Smith" {
		t.Errorf("Expected SN attribute to be 'Smith', got %v", fullAttrs["sn"])
	}

	if fullAttrs["mail"] == nil || fullAttrs["mail"][0] != "jane.smith@example.com" {
		t.Errorf("Expected mail attribute to be 'jane.smith@example.com', got %v", fullAttrs["mail"])
	}

	// Check POSIX attributes and object classes
	expectedPosixClasses := []string{"top", "inetOrgPerson", "posixAccount"}
	if !reflect.DeepEqual(fullAttrs["objectClass"], expectedPosixClasses) {
		t.Errorf("Expected objectClass to be %v, got %v", expectedPosixClasses, fullAttrs["objectClass"])
	}

	if fullAttrs["uidNumber"] == nil || fullAttrs["uidNumber"][0] != "1001" {
		t.Errorf("Expected uidNumber attribute to be '1001', got %v", fullAttrs["uidNumber"])
	}

	if fullAttrs["gidNumber"] == nil || fullAttrs["gidNumber"][0] != "1001" {
		t.Errorf("Expected gidNumber attribute to be '1001', got %v", fullAttrs["gidNumber"])
	}

	// homeDirectory should be set based on username field
	if fullAttrs["homeDirectory"] == nil || fullAttrs["homeDirectory"][0] != "/home/jsmith" {
		t.Errorf("Expected homeDirectory to be '/home/jsmith', got %v", fullAttrs["homeDirectory"])
	}
}

func TestGetSvcAcctAttributes(t *testing.T) {
	// Test a minimal service account
	minimalSvcAcct := &model.SvcAcct{
		Username:    "backup",
		CN:          "Backup Service",
		Description: "Service for backups",
	}
	minAttrs := GetSvcAcctAttributes(minimalSvcAcct)

	if minAttrs["cn"] == nil || minAttrs["cn"][0] != "Backup Service" {
		t.Errorf("Expected CN attribute to be 'Backup Service', got %v", minAttrs["cn"])
	}

	if minAttrs["description"] == nil || minAttrs["description"][0] != "Service for backups" {
		t.Errorf("Expected description attribute to be 'Service for backups', got %v", minAttrs["description"])
	}

	// Check object classes for non-POSIX service account
	expectedClasses := []string{"top", "inetOrgPerson", "account"}
	if !reflect.DeepEqual(minAttrs["objectClass"], expectedClasses) {
		t.Errorf("Expected objectClass to be %v, got %v", expectedClasses, minAttrs["objectClass"])
	}

	// Test a complete service account
	fullSvcAcct := &model.SvcAcct{
		Username:    "auth",
		CN:          "Auth Service",
		Description: "Authentication service",
		Mail:        "auth@example.com",
		Posix:       []int{1050, 1051},
	}
	fullAttrs := GetSvcAcctAttributes(fullSvcAcct)

	if fullAttrs["cn"] == nil || fullAttrs["cn"][0] != "Auth Service" {
		t.Errorf("Expected CN attribute to be 'Auth Service', got %v", fullAttrs["cn"])
	}

	if fullAttrs["description"] == nil || fullAttrs["description"][0] != "Authentication service" {
		t.Errorf("Expected description attribute to be 'Authentication service', got %v", fullAttrs["description"])
	}

	if fullAttrs["mail"] == nil || fullAttrs["mail"][0] != "auth@example.com" {
		t.Errorf("Expected mail attribute to be 'auth@example.com', got %v", fullAttrs["mail"])
	}

	// Check POSIX attributes and object classes
	expectedPosixClasses := []string{"top", "inetOrgPerson", "posixAccount"}
	if !reflect.DeepEqual(fullAttrs["objectClass"], expectedPosixClasses) {
		t.Errorf("Expected objectClass to be %v, got %v", expectedPosixClasses, fullAttrs["objectClass"])
	}

	if fullAttrs["uidNumber"] == nil || fullAttrs["uidNumber"][0] != "1050" {
		t.Errorf("Expected uidNumber attribute to be '1050', got %v", fullAttrs["uidNumber"])
	}

	if fullAttrs["gidNumber"] == nil || fullAttrs["gidNumber"][0] != "1051" {
		t.Errorf("Expected gidNumber attribute to be '1051', got %v", fullAttrs["gidNumber"])
	}

	// homeDirectory should be set based on username field
	if fullAttrs["homeDirectory"] == nil || fullAttrs["homeDirectory"][0] != "/home/auth" {
		t.Errorf("Expected homeDirectory to be '/home/auth', got %v", fullAttrs["homeDirectory"])
	}
}

func TestGetGroupAttributes(t *testing.T) {
	// Create a test configuration
	testConfig := &config.Config{
		LDAPEnforcer: config.LDAPEnforcerConfig{
			BindDN:        "cn=admin,dc=example,dc=com",
			PeopleBaseDN:  "ou=people,dc=example,dc=com",
			SvcAcctBaseDN: "ou=svcaccts,dc=example,dc=com",
			GroupBaseDN:   "ou=groups,dc=example,dc=com",
			ManagedOU:     "managed",
			Person: map[string]*model.Person{
				"john": {CN: "John Doe"},
				"jane": {CN: "Jane Smith"},
			},
			SvcAcct: map[string]*model.SvcAcct{
				"backup": {CN: "Backup Service", Description: "Backup service"},
			},
			Group: map[string]*model.Group{
				"admins": {
					Description: "Administrators",
					People:      []string{"john"},
					SvcAccts:    []string{"backup"},
				},
				"users": {
					Description: "Regular Users",
					People:      []string{"jane"},
				},
				"all": {
					Description: "All users",
					Groups:      []string{"admins", "users"},
				},
			},
		},
	}

	client := &Client{
		config: testConfig,
	}

	// Test a simple group
	usersGroup := testConfig.LDAPEnforcer.Group["users"]
	usersAttrs, err := client.GetGroupAttributes("users", usersGroup)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if usersAttrs["cn"] == nil || usersAttrs["cn"][0] != "users" {
		t.Errorf("Expected CN attribute to be 'users', got %v", usersAttrs["cn"])
	}

	if usersAttrs["description"] == nil || usersAttrs["description"][0] != "Regular Users" {
		t.Errorf("Expected description attribute to be 'Regular Users', got %v", usersAttrs["description"])
	}

	// Check that the group has the expected member
	if len(usersAttrs["member"]) != 1 {
		t.Fatalf("Expected 1 member, got %d", len(usersAttrs["member"]))
	}

	expectedMember := "uid=jane,ou=managed,ou=people,dc=example,dc=com"
	if usersAttrs["member"][0] != expectedMember {
		t.Errorf("Expected member to be '%s', got '%s'", expectedMember, usersAttrs["member"][0])
	}

	// Test a group with nested groups
	allGroup := testConfig.LDAPEnforcer.Group["all"]
	allAttrs, err := client.GetGroupAttributes("all", allGroup)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The "all" group should include members from both "admins" and "users" groups
	if len(allAttrs["member"]) != 3 {
		t.Fatalf("Expected 3 members, got %d", len(allAttrs["member"]))
	}
}
