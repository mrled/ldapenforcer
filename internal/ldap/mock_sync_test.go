package ldap

import (
	"testing"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/model"
)

// TestGroupUserReplacement tests a scenario where a group has a single user,
// the sync deletes that user, adds a new user, and updates the group.
// This test verifies that the group is NEVER deleted during the operation.
func TestGroupUserReplacement(t *testing.T) {
	// Create a test configuration with a group that has a single user
	testConfig := &config.Config{
		LDAPEnforcer: config.LDAPEnforcerConfig{
			URI:               "ldap://localhost:389",
			BindDN:            "cn=admin,dc=example,dc=com",
			Password:          "admin",
			EnforcedPeopleOU:  "ou=people,dc=example,dc=com",
			EnforcedSvcAcctOU: "ou=svcaccts,dc=example,dc=com",
			EnforcedGroupOU:   "ou=groups,dc=example,dc=com",
			Person: map[string]*model.Person{
				// We'll add "newuser" but not "olduser" (which will be deleted)
				"newuser": {
					CN:        "New User",
					GivenName: "New",
					SN:        "User",
					Mail:      "newuser@example.com",
				},
			},
			Group: map[string]*model.Group{
				"testgroup": {
					Description: "Test Group",
					People:      []string{"newuser"}, // Group now references the new user
				},
			},
		},
	}

	// Create a mock LDAP client
	mockClient := NewMockClient(testConfig)

	// Set up the "existing" LDAP state:
	// 1. "olduser" exists
	oldUserDN := "uid=olduser," + testConfig.LDAPEnforcer.EnforcedPeopleOU
	mockClient.Existing[oldUserDN] = true

	// 2. The group exists and contains olduser
	testGroupDN := "cn=testgroup," + testConfig.LDAPEnforcer.EnforcedGroupOU
	mockClient.Existing[testGroupDN] = true

	// Run the sync
	err := mockClient.SyncAll()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Analyze operations
	var groupDeleteOp bool
	var groupModifyOp bool
	var deleteOldUserOp bool
	var createNewUserOp bool

	for _, op := range mockClient.Operations {
		switch {
		case op.OpType == "delete" && op.Type == "group" && op.EntityID == "testgroup":
			groupDeleteOp = true
		case op.OpType == "modify" && op.Type == "group" && op.EntityID == "testgroup":
			groupModifyOp = true
		case op.OpType == "delete" && op.Type == "person" && op.EntityID == "olduser":
			deleteOldUserOp = true
		case op.OpType == "create" && op.Type == "person" && op.EntityID == "newuser":
			createNewUserOp = true
		}
	}

	// Verify operations
	if groupDeleteOp {
		t.Error("The group should NOT have been deleted during the sync operation")
	}

	if !groupModifyOp {
		t.Error("The group should have been modified to reference the new user")
	}

	if !deleteOldUserOp {
		t.Error("The old user should have been deleted")
	}

	if !createNewUserOp {
		t.Error("The new user should have been created")
	}

	// Verify operation order
	createNewUserFound := false
	groupModifyFound := false

	for _, op := range mockClient.Operations {
		if op.OpType == "create" && op.Type == "person" && op.EntityID == "newuser" {
			createNewUserFound = true
		} else if op.OpType == "modify" && op.Type == "group" && op.EntityID == "testgroup" {
			// Verify that new user was created before group modification
			if !createNewUserFound {
				t.Error("Group was modified before new user was created")
			}
			groupModifyFound = true
		} else if op.OpType == "delete" && op.Type == "person" && op.EntityID == "olduser" {
			// Verify that group was modified before old user was deleted
			if !groupModifyFound {
				t.Error("Old user was deleted before group was modified")
			}
		}
	}
}
