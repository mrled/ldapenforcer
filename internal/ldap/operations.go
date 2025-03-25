package ldap

import (
	"fmt"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/mrled/ldapenforcer/internal/model"
)

// PersonToDN converts a person UID to a DN
func (c *Client) PersonToDN(uid string) string {
	return fmt.Sprintf("uid=%s,%s",
		ldap.EscapeFilter(uid),
		c.config.LDAPEnforcer.EnforcedPeopleOU)
}

// SvcAcctToDN converts a service account UID to a DN
func (c *Client) SvcAcctToDN(uid string) string {
	return fmt.Sprintf("uid=%s,%s",
		ldap.EscapeFilter(uid),
		c.config.LDAPEnforcer.EnforcedSvcAcctOU)
}

// GroupToDN converts a group name to a DN
func (c *Client) GroupToDN(groupname string) string {
	return fmt.Sprintf("cn=%s,%s",
		ldap.EscapeFilter(groupname),
		c.config.LDAPEnforcer.EnforcedGroupOU)
}

// GetPersonAttributes converts a Person to LDAP attributes
func GetPersonAttributes(person *model.Person) map[string][]string {
	// Base object classes
	objectClasses := []string{"top", "inetOrgPerson", "nsMemberOf"}

	// Add either account or posixAccount based on POSIX status
	if person.IsPosix() {
		objectClasses = append(objectClasses, "posixAccount")
	} else {
		objectClasses = append(objectClasses, "account")
	}

	attrs := map[string][]string{
		"objectClass": objectClasses,
		"cn":          {person.CN},
		"sn":          {person.GetSN()},
	}

	// Add optional attributes if set
	if person.GivenName != "" {
		attrs["givenName"] = []string{person.GivenName}
	}
	if person.Mail != "" {
		attrs["mail"] = []string{person.Mail}
	}

	// Add POSIX attributes if set
	if person.IsPosix() {
		attrs["uidNumber"] = []string{strconv.Itoa(person.GetUIDNumber())}
		attrs["gidNumber"] = []string{strconv.Itoa(person.GetGIDNumber())}

		// Set homeDirectory based on username field
		if person.Username != "" {
			attrs["homeDirectory"] = []string{fmt.Sprintf("/home/%s", person.Username)}
		} else {
			attrs["homeDirectory"] = []string{"/nonexistent"}
		}

		attrs["loginShell"] = []string{"/bin/bash"}
	}

	return attrs
}

// GetSvcAcctAttributes converts a SvcAcct to LDAP attributes
func GetSvcAcctAttributes(svcacct *model.SvcAcct) map[string][]string {
	// Base object classes
	objectClasses := []string{"top", "inetOrgPerson", "nsMemberOf"}

	// Add either account or posixAccount based on POSIX status
	if svcacct.IsPosix() {
		objectClasses = append(objectClasses, "posixAccount")
	} else {
		objectClasses = append(objectClasses, "account")
	}

	attrs := map[string][]string{
		"objectClass": objectClasses,
		"cn":          {svcacct.CN},
		"description": {svcacct.Description},
		"sn":          {svcacct.Username}, // Set sn to username for inetOrgPerson compliance
	}

	// Add optional attributes if set
	if svcacct.Mail != "" {
		attrs["mail"] = []string{svcacct.Mail}
	}

	// Add POSIX attributes if set
	if svcacct.IsPosix() {
		attrs["uidNumber"] = []string{strconv.Itoa(svcacct.GetUIDNumber())}
		attrs["gidNumber"] = []string{strconv.Itoa(svcacct.GetGIDNumber())}

		// Set homeDirectory based on username field
		if svcacct.Username != "" {
			attrs["homeDirectory"] = []string{fmt.Sprintf("/home/%s", svcacct.Username)}
		} else {
			attrs["homeDirectory"] = []string{"/nonexistent"}
		}

		attrs["loginShell"] = []string{"/usr/sbin/nologin"}
	}

	return attrs
}

// GetGroupAttributes converts a Group to LDAP attributes
func (c *Client) GetGroupAttributes(groupname string, group *model.Group) (map[string][]string, error) {
	attrs := map[string][]string{
		"objectClass": {"top", "groupOfNames"},
		"cn":          {groupname},
		"description": {group.Description},
	}

	// Get all members including from nested groups
	members, err := model.GetGroupMembers(
		groupname,
		c.config.LDAPEnforcer.Group,
		c.config.LDAPEnforcer.Person,
		c.config.LDAPEnforcer.SvcAcct,
		c.config.LDAPEnforcer.EnforcedPeopleOU,
		c.config.LDAPEnforcer.EnforcedSvcAcctOU,
		c.config.LDAPEnforcer.EnforcedGroupOU,
	)
	if err != nil {
		return nil, err
	}

	// Add all member DNs
	var memberDNs []string
	for _, member := range members {
		memberDNs = append(memberDNs, member.DN)
	}

	// We require at least one member for a valid group
	if len(memberDNs) == 0 {
		return nil, fmt.Errorf("group has no members after resolving: %s", groupname)
	}

	attrs["member"] = memberDNs

	// Add POSIX attributes if set
	if group.IsPosix() {
		attrs["objectClass"] = append(attrs["objectClass"], "posixGroup")
		attrs["gidNumber"] = []string{strconv.Itoa(group.PosixGidNumber)}
	}

	return attrs, nil
}

// EnsureManagedOUsExist ensures that all required OUs for managed objects exist
func (c *Client) EnsureManagedOUsExist() error {
	// Ensure people OU exists
	err := c.EnsureOUExists(c.config.LDAPEnforcer.EnforcedPeopleOU)
	if err != nil {
		return fmt.Errorf("failed to ensure people OU exists: %w", err)
	}

	// Ensure service account OU exists
	err = c.EnsureOUExists(c.config.LDAPEnforcer.EnforcedSvcAcctOU)
	if err != nil {
		return fmt.Errorf("failed to ensure service account OU exists: %w", err)
	}

	// Ensure group OU exists
	err = c.EnsureOUExists(c.config.LDAPEnforcer.EnforcedGroupOU)
	if err != nil {
		return fmt.Errorf("failed to ensure group OU exists: %w", err)
	}

	return nil
}

// SyncPerson ensures that a person in LDAP matches the configuration
func (c *Client) SyncPerson(uid string, person *model.Person) error {
	dn := c.PersonToDN(uid)
	exists, err := c.EntryExists(dn)
	if err != nil {
		return err
	}

	// Set the Username field with the uid if it's not already set
	if person.Username == "" {
		person.Username = uid
	}

	attrs := GetPersonAttributes(person)

	if exists {
		logging.LDAPProtocolLogger.Trace("Updating person: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		logging.LDAPProtocolLogger.Trace("Creating person: %s", dn)
		// Add the uid attribute which is required
		attrs["uid"] = []string{uid}
		return c.CreateEntry(dn, attrs)
	}
}

// SyncSvcAcct ensures that a service account in LDAP matches the configuration
func (c *Client) SyncSvcAcct(uid string, svcacct *model.SvcAcct) error {
	dn := c.SvcAcctToDN(uid)
	exists, err := c.EntryExists(dn)
	if err != nil {
		return err
	}

	// Set the Username field with the uid if it's not already set
	if svcacct.Username == "" {
		svcacct.Username = uid
	}

	attrs := GetSvcAcctAttributes(svcacct)

	if exists {
		logging.LDAPProtocolLogger.Trace("Updating service account: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		logging.LDAPProtocolLogger.Trace("Creating service account: %s", dn)
		// Add the uid attribute which is required
		attrs["uid"] = []string{uid}
		return c.CreateEntry(dn, attrs)
	}
}

// GetExistingEntries returns a map of DNs to entities for the given OU and type
func (c *Client) GetExistingEntries(ou string, entryType string) (map[string]string, error) {
	// Prepare the search filter based on entry type
	var filter string
	switch entryType {
	case "person":
		filter = "(objectClass=inetOrgPerson)"
	case "svcacct":
		filter = "(objectClass=inetOrgPerson)"
	case "group":
		filter = "(objectClass=groupOfNames)"
	default:
		return nil, fmt.Errorf("unsupported entry type: %s", entryType)
	}

	// Search for all entries in the OU
	searchRequest := ldap.NewSearchRequest(
		ou,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		filter,
		[]string{"dn"},
		nil,
	)

	searchResult, err := c.conn.Search(searchRequest)
	if err != nil {
		// If the error is "No Such Object", it means the OU doesn't exist yet
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("failed to search for entries in %s: %w", ou, err)
	}

	// Create a map of DN to DN (using DN as both key and value for simplicity)
	entries := make(map[string]string)
	for _, entry := range searchResult.Entries {
		entries[entry.DN] = entry.DN
	}

	return entries, nil
}

// getGroupDependencies returns a list of groups that this group depends on
// and a list of unresolvable member UIDs
func (c *Client) getGroupDependencies(groupname string, processedGroups map[string]bool) ([]string, []string) {
	// Mark this group as visited to avoid infinite recursion
	processedGroups[groupname] = true

	// Get the group
	group, ok := c.config.LDAPEnforcer.Group[groupname]
	if !ok {
		return nil, nil
	}

	// Track dependencies and unresolvable members
	var dependencies []string
	var unresolvableMembers []string

	// Check people members
	for _, uid := range group.People {
		if _, ok := c.config.LDAPEnforcer.Person[uid]; !ok {
			unresolvableMembers = append(unresolvableMembers, uid)
		}
	}

	// Check service account members
	for _, uid := range group.SvcAccts {
		if _, ok := c.config.LDAPEnforcer.SvcAcct[uid]; !ok {
			unresolvableMembers = append(unresolvableMembers, uid)
		}
	}

	// Check group members and recursively build dependencies
	for _, nestedGroupName := range group.Groups {
		// Add as dependency
		dependencies = append(dependencies, nestedGroupName)

		// If we haven't processed this nested group yet, get its dependencies too
		if !processedGroups[nestedGroupName] {
			nestedDeps, nestedUnres := c.getGroupDependencies(nestedGroupName, processedGroups)
			dependencies = append(dependencies, nestedDeps...)
			unresolvableMembers = append(unresolvableMembers, nestedUnres...)
		}
	}

	return dependencies, unresolvableMembers
}

// topologicalSortGroups returns a slice of group names in topological order
// (dependencies first, dependents later)
func (c *Client) topologicalSortGroups(deps map[string][]string) []string {
	// First, get all groups (including those with no dependencies)
	allGroups := make(map[string]bool)
	for groupname := range c.config.LDAPEnforcer.Group {
		allGroups[groupname] = true
	}
	for groupname, groupDeps := range deps {
		allGroups[groupname] = true
		for _, dep := range groupDeps {
			allGroups[dep] = true
		}
	}

	// Now, build a proper adjacency list from the dependencies
	graph := make(map[string][]string)
	for group := range allGroups {
		graph[group] = []string{}
	}
	for group, groupDeps := range deps {
		// Add all dependencies at once
		graph[group] = append(graph[group], groupDeps...)
	}

	// Perform topological sort
	visited := make(map[string]bool)
	temp := make(map[string]bool) // For cycle detection
	var order []string

	// Visit function for DFS
	var visit func(string)
	visit = func(node string) {
		// If we've already processed this node, skip
		if visited[node] {
			return
		}

		// If we're currently processing this node, we have a cycle
		if temp[node] {
			// We have a cycle, but we should still proceed
			logging.DefaultLogger.Debug("Cyclic dependency detected involving group: %s", node)
			return
		}

		temp[node] = true

		// Visit all dependencies before adding this node
		for _, dep := range graph[node] {
			visit(dep)
		}

		temp[node] = false
		visited[node] = true
		order = append(order, node)
	}

	// Visit each node
	for node := range graph {
		if !visited[node] {
			visit(node)
		}
	}

	// Reverse the order to get dependencies first
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order
}

// SyncGroup ensures that a group in LDAP matches the configuration
func (c *Client) SyncGroup(groupname string, group *model.Group) error {
	dn := c.GroupToDN(groupname)
	exists, err := c.EntryExists(dn)
	if err != nil {
		return err
	}

	attrs, err := c.GetGroupAttributes(groupname, group)
	if err != nil {
		// If the error indicates an empty group
		if err.Error() == fmt.Sprintf("group has no members after resolving: %s", groupname) {
			if exists {
				logging.DefaultLogger.Warn("Deleting memberless group %s (add at least one member)", groupname)
				return c.DeleteEntry(dn)
			} else {
				logging.DefaultLogger.Warn("Refusing to create memberless group %s (add at least one member)", groupname)
			}
			return nil
		}
		return err
	}

	if exists {
		// Update existing group
		logging.LDAPProtocolLogger.Trace("Updating group: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		// Create new group
		logging.LDAPProtocolLogger.Trace("Creating group: %s", dn)
		return c.CreateEntry(dn, attrs)
	}
}

// SyncAll synchronizes all configured entities with LDAP using a DAG approach
func (c *Client) SyncAll() error {
	// Ensure all required OUs exist
	err := c.EnsureManagedOUsExist()
	if err != nil {
		return err
	}

	// First, get existing entities to determine what needs to be added, modified, and deleted
	existingPeople, err := c.GetExistingEntries(c.config.LDAPEnforcer.EnforcedPeopleOU, "person")
	if err != nil {
		return fmt.Errorf("failed to get existing people: %w", err)
	}

	existingSvcAccts, err := c.GetExistingEntries(c.config.LDAPEnforcer.EnforcedSvcAcctOU, "svcacct")
	if err != nil {
		return fmt.Errorf("failed to get existing service accounts: %w", err)
	}

	existingGroups, err := c.GetExistingEntries(c.config.LDAPEnforcer.EnforcedGroupOU, "group")
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
	for uid, person := range c.config.LDAPEnforcer.Person {
		dn := c.PersonToDN(uid)
		if _, exists := existingPeople[dn]; exists {
			peopleToModify[uid] = person
			delete(existingPeople, dn) // Remove from existing so we know what to delete
		} else {
			peopleToAdd[uid] = person
		}
	}

	for uid, svcacct := range c.config.LDAPEnforcer.SvcAcct {
		dn := c.SvcAcctToDN(uid)
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
	for groupname, group := range c.config.LDAPEnforcer.Group {
		dn := c.GroupToDN(groupname)
		if _, exists := existingGroups[dn]; exists {
			groupsToModify[groupname] = group
			delete(existingGroups, dn) // Remove from existing so we know what to delete
		} else {
			groupsToAdd[groupname] = group
		}

		// Collect dependencies for this group
		deps, unresMem := c.getGroupDependencies(groupname, processedGroups)
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
		err := c.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to add person %s: %w", uid, err)
		}
	}

	for uid, person := range peopleToModify {
		err := c.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to modify person %s: %w", uid, err)
		}
	}

	for uid, svcacct := range svcAcctsToAdd {
		err := c.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to add service account %s: %w", uid, err)
		}
	}

	for uid, svcacct := range svcAcctsToModify {
		err := c.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to modify service account %s: %w", uid, err)
		}
	}

	// 2. Then add and modify groups in dependency order (leaf groups first)
	// Get groups in order
	groupOrder := c.topologicalSortGroups(groupDeps)

	// Add and modify groups in order (leaf groups first)
	for _, groupname := range groupOrder {
		// Check if it's in our add or modify list
		if group, ok := groupsToAdd[groupname]; ok {
			err := c.SyncGroup(groupname, group)
			if err != nil {
				return fmt.Errorf("failed to add group %s: %w", groupname, err)
			}
		} else if group, ok := groupsToModify[groupname]; ok {
			err := c.SyncGroup(groupname, group)
			if err != nil {
				return fmt.Errorf("failed to modify group %s: %w", groupname, err)
			}
		}
	}

	// 3. Finally delete entities (groups first)
	for _, dn := range groupsToDelete {
		err := c.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete group %s: %w", dn, err)
		}
	}

	for _, dn := range peopleToDelete {
		err := c.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete person %s: %w", dn, err)
		}
	}

	for _, dn := range svcAcctsToDelete {
		err := c.DeleteEntry(dn)
		if err != nil {
			return fmt.Errorf("failed to delete service account %s: %w", dn, err)
		}
	}

	return nil
}
