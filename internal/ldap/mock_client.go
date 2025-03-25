package ldap

import (
	"fmt"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/mrled/ldapenforcer/internal/model"
)

// MockOperation represents a mock LDAP operation for testing
type MockOperation struct {
	OpType   string // "create", "modify", "delete"
	DN       string
	EntityID string
	Type     string // "person", "svcacct", "group"
}

// MockClient is a mock LDAP client for testing
type MockClient struct {
	BaseClient
	Operations []MockOperation
	Existing   map[string]bool // map of DNs that "exist" in the mock LDAP server
}

// NewMockClient creates a new mock LDAP client
func NewMockClient(cfg *config.Config) *MockClient {
	return &MockClient{
		BaseClient: BaseClient{
			config: cfg,
		},
		Operations: []MockOperation{},
		Existing:   make(map[string]bool),
	}
}

// Close is a no-op for the mock client
func (m *MockClient) Close() error {
	return nil
}

// EntryExists checks if an entry exists in the mock LDAP server
func (m *MockClient) EntryExists(dn string) (bool, error) {
	return m.Existing[dn], nil
}

// CreateEntry records a create operation and marks the entry as existing
func (m *MockClient) CreateEntry(dn string, attrs map[string][]string) error {
	entityType, entityID := getEntityTypeAndID(dn)

	m.Operations = append(m.Operations, MockOperation{
		OpType:   "create",
		DN:       dn,
		EntityID: entityID,
		Type:     entityType,
	})
	m.Existing[dn] = true
	return nil
}

// ModifyEntry records a modify operation
func (m *MockClient) ModifyEntry(dn string, attrs map[string][]string, modType int) error {
	entityType, entityID := getEntityTypeAndID(dn)

	m.Operations = append(m.Operations, MockOperation{
		OpType:   "modify",
		DN:       dn,
		EntityID: entityID,
		Type:     entityType,
	})
	return nil
}

// DeleteEntry records a delete operation and marks the entry as non-existing
func (m *MockClient) DeleteEntry(dn string) error {
	entityType, entityID := getEntityTypeAndID(dn)

	m.Operations = append(m.Operations, MockOperation{
		OpType:   "delete",
		DN:       dn,
		EntityID: entityID,
		Type:     entityType,
	})
	m.Existing[dn] = false
	return nil
}

// GetExistingEntries returns a map of DNs that exist in the specified OU
func (m *MockClient) GetExistingEntries(ou string, entryType string) (map[string]string, error) {
	result := make(map[string]string)

	// Return only entries that match the requested OU and type and exist
	for dn, exists := range m.Existing {
		if exists && isDNInOU(dn, ou) {
			result[dn] = dn
		}
	}

	return result, nil
}

// EnsureManagedOUsExist ensures that all required OUs exist
func (m *MockClient) EnsureManagedOUsExist() error {
	// Just mark OUs as "existing"
	m.Existing[m.config.LDAPEnforcer.EnforcedPeopleOU] = true
	m.Existing[m.config.LDAPEnforcer.EnforcedSvcAcctOU] = true
	m.Existing[m.config.LDAPEnforcer.EnforcedGroupOU] = true
	return nil
}

// EnsureOUExists ensures that an OU exists
func (m *MockClient) EnsureOUExists(ou string) error {
	m.Existing[ou] = true
	return nil
}

// SyncPerson ensures that a person in LDAP matches the configuration
func (m *MockClient) SyncPerson(uid string, person *model.Person) error {
	dn := m.PersonToDN(uid)
	exists, _ := m.EntryExists(dn)

	if exists {
		return m.ModifyEntry(dn, nil, ldap.ReplaceAttribute)
	} else {
		return m.CreateEntry(dn, nil)
	}
}

// SyncSvcAcct ensures that a service account in LDAP matches the configuration
func (m *MockClient) SyncSvcAcct(uid string, svcacct *model.SvcAcct) error {
	dn := m.SvcAcctToDN(uid)
	exists, _ := m.EntryExists(dn)

	if exists {
		return m.ModifyEntry(dn, nil, ldap.ReplaceAttribute)
	} else {
		return m.CreateEntry(dn, nil)
	}
}

// SyncGroup ensures that a group in LDAP matches the configuration
func (m *MockClient) SyncGroup(groupname string, group *model.Group) error {
	dn := m.GroupToDN(groupname)
	exists, _ := m.EntryExists(dn)

	// Get all members
	members, _ := model.GetGroupMembers(
		groupname,
		m.config.LDAPEnforcer.Group,
		m.config.LDAPEnforcer.Person,
		m.config.LDAPEnforcer.SvcAcct,
		m.config.LDAPEnforcer.EnforcedPeopleOU,
		m.config.LDAPEnforcer.EnforcedSvcAcctOU,
		m.config.LDAPEnforcer.EnforcedGroupOU,
	)

	// If group has no members, handle appropriately
	if len(members) == 0 {
		if exists {
			return m.DeleteEntry(dn)
		}
		return nil
	}

	if exists {
		return m.ModifyEntry(dn, nil, ldap.ReplaceAttribute)
	} else {
		return m.CreateEntry(dn, nil)
	}
}

// SyncAll synchronizes all configured entities with LDAP
func (m *MockClient) SyncAll() error {
	// Ensure all required OUs exist
	err := m.EnsureManagedOUsExist()
	if err != nil {
		return err
	}

	// First, get existing entities to determine what needs to be added, modified, and deleted
	existingPeople, err := m.GetExistingEntries(m.config.LDAPEnforcer.EnforcedPeopleOU, "person")
	if err != nil {
		return fmt.Errorf("failed to get existing people: %w", err)
	}

	existingSvcAccts, err := m.GetExistingEntries(m.config.LDAPEnforcer.EnforcedSvcAcctOU, "svcacct")
	if err != nil {
		return fmt.Errorf("failed to get existing service accounts: %w", err)
	}

	existingGroups, err := m.GetExistingEntries(m.config.LDAPEnforcer.EnforcedGroupOU, "group")
	if err != nil {
		return fmt.Errorf("failed to get existing groups: %w", err)
	}

	// Build a DAG of all entities to determine proper operation order
	peopleToAdd := make(map[string]*model.Person)
	peopleToModify := make(map[string]*model.Person)
	peopleToDelete := make(map[string]string) // DN as value

	svcAcctsToAdd := make(map[string]*model.SvcAcct)
	svcAcctsToModify := make(map[string]*model.SvcAcct)
	svcAcctsToDelete := make(map[string]string) // DN as value

	groupsToAdd := make(map[string]*model.Group)
	groupsToModify := make(map[string]*model.Group)
	groupsToDelete := make(map[string]string) // DN as value

	// Determine people and service accounts to add, modify, or delete
	for uid, person := range m.config.LDAPEnforcer.Person {
		dn := m.PersonToDN(uid)
		if _, exists := existingPeople[dn]; exists {
			peopleToModify[uid] = person
			delete(existingPeople, dn) // Remove from existing so we know what to delete
		} else {
			peopleToAdd[uid] = person
		}
	}

	for uid, svcacct := range m.config.LDAPEnforcer.SvcAcct {
		dn := m.SvcAcctToDN(uid)
		if _, exists := existingSvcAccts[dn]; exists {
			svcAcctsToModify[uid] = svcacct
			delete(existingSvcAccts, dn) // Remove from existing so we know what to delete
		} else {
			svcAcctsToAdd[uid] = svcacct
		}
	}

	// Any remaining entries in existingPeople/existingSvcAccts are not in config and should be deleted
	for dn := range existingPeople {
		peopleToDelete[dn] = dn
	}
	for dn := range existingSvcAccts {
		svcAcctsToDelete[dn] = dn
	}

	// Build the group dependency graph
	// Keep track of processed groups to avoid cycles
	processedGroups := make(map[string]bool)
	groupDeps := make(map[string][]string)           // Map of groupname -> groups it depends on
	unresolvableMembers := make(map[string][]string) // Map of groupname -> unresolvable member UIDs

	// Process all groups to determine dependencies and operation type
	for groupname, group := range m.config.LDAPEnforcer.Group {
		dn := m.GroupToDN(groupname)
		if _, exists := existingGroups[dn]; exists {
			groupsToModify[groupname] = group
			delete(existingGroups, dn) // Remove from existing so we know what to delete
		} else {
			groupsToAdd[groupname] = group
		}

		// Collect dependencies for this group
		deps, unresMem := m.getGroupDependencies(groupname, processedGroups)
		if len(deps) > 0 {
			groupDeps[groupname] = deps
		}
		if len(unresMem) > 0 {
			unresolvableMembers[groupname] = unresMem
		}
	}

	// Any remaining entries in existingGroups are not in config and should be deleted
	for dn := range existingGroups {
		groupsToDelete[dn] = dn
	}

	// Log any unresolvable members
	for groupname, members := range unresolvableMembers {
		for _, member := range members {
			logging.DefaultLogger.Warn("Unresolvable member %s in group %s", member, groupname)
		}
	}

	// Execute operations in the correct order:
	// 1. First add and modify people and service accounts
	for uid, person := range peopleToAdd {
		err := m.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to add person %s: %w", uid, err)
		}
	}

	for uid, person := range peopleToModify {
		err := m.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to modify person %s: %w", uid, err)
		}
	}

	for uid, svcacct := range svcAcctsToAdd {
		err := m.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to add service account %s: %w", uid, err)
		}
	}

	for uid, svcacct := range svcAcctsToModify {
		err := m.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to modify service account %s: %w", uid, err)
		}
	}

	// 2. Then add and modify groups in dependency order (leaf groups first)
	// Get groups in order
	groupOrder := m.topologicalSortGroups(groupDeps)

	// Add and modify groups in order (leaf groups first)
	for _, groupname := range groupOrder {
		// Check if it's in our add or modify list
		if group, ok := groupsToAdd[groupname]; ok {
			err := m.SyncGroup(groupname, group)
			if err != nil {
				return fmt.Errorf("failed to add group %s: %w", groupname, err)
			}
		} else if group, ok := groupsToModify[groupname]; ok {
			err := m.SyncGroup(groupname, group)
			if err != nil {
				return fmt.Errorf("failed to modify group %s: %w", groupname, err)
			}
		}
	}

	// 3. Finally delete entities (groups first)
	for _, dn := range groupsToDelete {
		err := m.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete group %s: %w", dn, err)
		}
	}

	for _, dn := range peopleToDelete {
		err := m.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete person %s: %w", dn, err)
		}
	}

	for _, dn := range svcAcctsToDelete {
		err := m.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete service account %s: %w", dn, err)
		}
	}

	return nil
}

// Helper function to determine entity type and ID from a DN
func getEntityTypeAndID(dn string) (string, string) {
	if dn == "" {
		return "", ""
	}

	// Extract the first part of the DN (e.g., "uid=john" or "cn=admins")
	firstPart := dn
	if commaIdx := strings.Index(dn, ","); commaIdx != -1 {
		firstPart = dn[:commaIdx]
	}

	// Check if it's a person/svcacct (uid=) or group (cn=)
	if len(firstPart) > 4 && firstPart[:4] == "uid=" {
		return "person", firstPart[4:]
	} else if len(firstPart) > 3 && firstPart[:3] == "cn=" {
		return "group", firstPart[3:]
	}

	return "unknown", ""
}

// Helper function to check if a DN is in an OU
func isDNInOU(dn, ou string) bool {
	if dn == "" || ou == "" {
		return false
	}

	// Check if dn ends with ou
	return len(dn) > len(ou) && dn[len(dn)-len(ou):] == ou
}
