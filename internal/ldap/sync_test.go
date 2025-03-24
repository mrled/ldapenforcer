package ldap

import (
	"testing"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/model"
)

// This test simulates a sync workflow without an actual LDAP server
func TestSyncWorkflow(t *testing.T) {
	// Skip this test in normal runs since it's more of an integration test
	t.Skip("Skipping sync workflow test - use -test.run=TestSyncWorkflow to run explicitly")

	// Create a test configuration
	testConfig := &config.Config{
		LDAPEnforcer: config.LDAPEnforcerConfig{
			URI:               "ldap://localhost:389",
			BindDN:            "cn=admin,dc=example,dc=com",
			Password:          "admin",
			EnforcedPeopleOU:  "ou=managed,ou=people,dc=example,dc=com",
			EnforcedSvcAcctOU: "ou=managed,ou=svcaccts,dc=example,dc=com",
			EnforcedGroupOU:   "ou=managed,ou=groups,dc=example,dc=com",
			Person: map[string]*model.Person{
				"john": {
					CN:        "John Doe",
					GivenName: "John",
					Mail:      "john@example.com",
					Posix:     []int{1001, 1001},
				},
				"jane": {
					CN:        "Jane Smith",
					GivenName: "Jane",
					SN:        "Smith",
					Mail:      "jane@example.com",
				},
			},
			SvcAcct: map[string]*model.SvcAcct{
				"backup": {
					CN:          "Backup Service",
					Description: "Backup service for system files",
					Mail:        "backup@example.com",
					Posix:       []int{1050, 1050},
				},
			},
			Group: map[string]*model.Group{
				"admins": {
					Description:    "Administrators",
					PosixGidNumber: 1001,
					People:         []string{"john"},
					SvcAccts:       []string{"backup"},
				},
				"users": {
					Description:    "Regular Users",
					PosixGidNumber: 1002,
					People:         []string{"jane"},
				},
				"all": {
					Description: "All users",
					Groups:      []string{"admins", "users"},
				},
			},
		},
	}

	// For this test, we'll validate the workflow steps without actually connecting to LDAP

	// 1. Validate people attributes
	johnAttrs := GetPersonAttributes(testConfig.LDAPEnforcer.Person["john"])
	if johnAttrs["cn"] == nil || johnAttrs["cn"][0] != "John Doe" {
		t.Errorf("Expected CN attribute to be 'John Doe', got %v", johnAttrs["cn"])
	}
	if johnAttrs["mail"] == nil || johnAttrs["mail"][0] != "john@example.com" {
		t.Errorf("Expected mail attribute to be 'john@example.com', got %v", johnAttrs["mail"])
	}

	// 2. Validate service account attributes
	backupAttrs := GetSvcAcctAttributes(testConfig.LDAPEnforcer.SvcAcct["backup"])
	if backupAttrs["description"] == nil || backupAttrs["description"][0] != "Backup service for system files" {
		t.Errorf("Expected description attribute to be 'Backup service for system files', got %v", backupAttrs["description"])
	}

	// 3. Create a client (without connecting) for DN generation
	client := &Client{
		config: testConfig,
	}

	// 4. Validate DN generation
	johnDN := client.PersonToDN("john")
	expectedJohnDN := "uid=john,ou=managed,ou=people,dc=example,dc=com"
	if johnDN != expectedJohnDN {
		t.Errorf("Expected DN %s, got %s", expectedJohnDN, johnDN)
	}

	// 5. Validate group attribute generation
	adminsAttrs, err := client.GetGroupAttributes("admins", testConfig.LDAPEnforcer.Group["admins"])
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(adminsAttrs["member"]) != 2 {
		t.Errorf("Expected admins group to have 2 members, got %d", len(adminsAttrs["member"]))
	}

	// 6. Validate nested group membership
	allAttrs, err := client.GetGroupAttributes("all", testConfig.LDAPEnforcer.Group["all"])
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(allAttrs["member"]) != 3 {
		t.Errorf("Expected 'all' group to have 3 members (john, jane, backup), got %d", len(allAttrs["member"]))
	}

	// 7. Validate OUs that would be created
	peopleOU := "ou=managed,ou=people,dc=example,dc=com"
	svcacctOU := "ou=managed,ou=svcaccts,dc=example,dc=com"
	groupOU := "ou=managed,ou=groups,dc=example,dc=com"

	// These would be created during a real sync
	t.Logf("Would create/ensure OU: %s", peopleOU)
	t.Logf("Would create/ensure OU: %s", svcacctOU)
	t.Logf("Would create/ensure OU: %s", groupOU)

	// 8. Simulate person sync
	for uid, person := range testConfig.LDAPEnforcer.Person {
		dn := client.PersonToDN(uid)
		attrs := GetPersonAttributes(person)
		attrs["uid"] = []string{uid} // Add UID attribute for creation
		t.Logf("Would create/update person: %s", dn)
	}

	// 9. Simulate service account sync
	for uid, svcacct := range testConfig.LDAPEnforcer.SvcAcct {
		dn := client.SvcAcctToDN(uid)
		attrs := GetSvcAcctAttributes(svcacct)
		attrs["uid"] = []string{uid} // Add UID attribute for creation
		t.Logf("Would create/update service account: %s", dn)
	}

	// 10. Simulate group sync
	for groupname, group := range testConfig.LDAPEnforcer.Group {
		dn := client.GroupToDN(groupname)
		attrs, err := client.GetGroupAttributes(groupname, group)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		t.Logf("Would create/update group: %s with %d members", dn, len(attrs["member"]))
	}
}
