package model

import (
	"testing"
)

func TestGetGroupMembers(t *testing.T) {
	// Create test people
	people := map[string]*Person{
		"user1": {
			CN:   "User One",
			Mail: "user1@example.com",
			Posix: []int{1001, 1001},
		},
		"user2": {
			CN:   "User Two",
			Mail: "user2@example.com",
			Posix: []int{1002, 1002},
		},
	}
	
	// Create test service accounts
	svcaccts := map[string]*SvcAcct{
		"svc1": {
			CN:          "Service One",
			Description: "Service account 1",
		},
		"svc2": {
			CN:          "Service Two",
			Description: "Service account 2",
			Posix:       []int{2002},
		},
	}
	
	// Create test groups
	groups := map[string]*Group{
		"group1": {
			Description:    "Group 1",
			PosixGidNumber: 3001,
			People:         []string{"user1"},
			SvcAccts:       []string{"svc1"},
			Groups:         []string{},
		},
		"group2": {
			Description:    "Group 2",
			PosixGidNumber: 3002,
			People:         []string{"user2"},
			SvcAccts:       []string{"svc2"},
			Groups:         []string{},
		},
		"nestedgroup": {
			Description:    "Nested Group",
			PosixGidNumber: 3003,
			People:         []string{},
			SvcAccts:       []string{},
			Groups:         []string{"group1", "group2"},
		},
		"cyclicgroup1": {
			Description:    "Cyclic Group 1",
			PosixGidNumber: 3004,
			People:         []string{},
			SvcAccts:       []string{},
			Groups:         []string{"cyclicgroup2"},
		},
		"cyclicgroup2": {
			Description:    "Cyclic Group 2",
			PosixGidNumber: 3005,
			People:         []string{},
			SvcAccts:       []string{},
			Groups:         []string{"cyclicgroup1"},
		},
	}
	
	// Directory structure
	peopleBaseDN := "ou=people,dc=example,dc=com"
	svcacctBaseDN := "ou=svcaccts,dc=example,dc=com"
	groupBaseDN := "ou=groups,dc=example,dc=com"
	managedOU := "enforced"
	
	// Test getting members of group1
	group1Members, err := GetGroupMembers("group1", groups, people, svcaccts, peopleBaseDN, svcacctBaseDN, groupBaseDN, managedOU)
	if err != nil {
		t.Fatalf("Error getting group1 members: %v", err)
	}
	if len(group1Members) != 2 {
		t.Errorf("Expected 2 members in group1, got %d", len(group1Members))
	}

	// Test getting members of the nested group
	nestedMembers, err := GetGroupMembers("nestedgroup", groups, people, svcaccts, peopleBaseDN, svcacctBaseDN, groupBaseDN, managedOU)
	if err != nil {
		t.Fatalf("Error getting nested group members: %v", err)
	}
	if len(nestedMembers) != 4 {
		t.Errorf("Expected 4 members in nested group, got %d", len(nestedMembers))
	}
	
	// Test cyclic group references (should not cause infinite recursion)
	cyclicMembers, err := GetGroupMembers("cyclicgroup1", groups, people, svcaccts, peopleBaseDN, svcacctBaseDN, groupBaseDN, managedOU)
	if err != nil {
		t.Fatalf("Error getting cyclic group members: %v", err)
	}
	if len(cyclicMembers) != 0 {
		t.Errorf("Expected 0 members in cyclic group, got %d", len(cyclicMembers))
	}
}