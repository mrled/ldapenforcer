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

// SyncAll synchronizes all configured entities with LDAP
func (c *Client) SyncAll() error {
	// Ensure all required OUs exist
	err := c.EnsureManagedOUsExist()
	if err != nil {
		return err
	}

	// Sync all people
	for uid, person := range c.config.LDAPEnforcer.Person {
		err := c.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to sync person %s: %w", uid, err)
		}
	}

	// Sync all service accounts
	for uid, svcacct := range c.config.LDAPEnforcer.SvcAcct {
		err := c.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to sync service account %s: %w", uid, err)
		}
	}

	// Sync all groups
	for groupname, group := range c.config.LDAPEnforcer.Group {
		err := c.SyncGroup(groupname, group)
		if err != nil {
			return fmt.Errorf("failed to sync group %s: %w", groupname, err)
		}
	}

	return nil
}
