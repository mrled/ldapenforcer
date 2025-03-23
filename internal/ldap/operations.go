package ldap

import (
	"fmt"
	"log"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/model"
)

// PersonToDN converts a person UID to a DN
func (c *Client) PersonToDN(uid string) string {
	return fmt.Sprintf("uid=%s,ou=%s,%s",
		ldap.EscapeFilter(uid),
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.PeopleBaseDN)
}

// SvcAcctToDN converts a service account UID to a DN
func (c *Client) SvcAcctToDN(uid string) string {
	return fmt.Sprintf("uid=%s,ou=%s,%s",
		ldap.EscapeFilter(uid),
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.SvcAcctBaseDN)
}

// GroupToDN converts a group name to a DN
func (c *Client) GroupToDN(groupname string) string {
	return fmt.Sprintf("cn=%s,ou=%s,%s",
		ldap.EscapeFilter(groupname),
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.GroupBaseDN)
}

// GetPersonAttributes converts a Person to LDAP attributes
func GetPersonAttributes(person *model.Person) map[string][]string {
	attrs := map[string][]string{
		"objectClass": {"top", "person", "organizationalPerson", "inetOrgPerson"},
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
		attrs["objectClass"] = append(attrs["objectClass"], "posixAccount")
		attrs["uidNumber"] = []string{strconv.Itoa(person.GetUIDNumber())}
		attrs["gidNumber"] = []string{strconv.Itoa(person.GetGIDNumber())}
		attrs["homeDirectory"] = []string{fmt.Sprintf("/home/%s", person.CN)}
		attrs["loginShell"] = []string{"/bin/bash"}
	}

	return attrs
}

// GetSvcAcctAttributes converts a SvcAcct to LDAP attributes
func GetSvcAcctAttributes(svcacct *model.SvcAcct) map[string][]string {
	attrs := map[string][]string{
		"objectClass": {"top", "account", "simpleSecurityObject"},
		"cn":          {svcacct.CN},
		"description": {svcacct.Description},
	}

	// Add optional attributes if set
	if svcacct.Mail != "" {
		attrs["mail"] = []string{svcacct.Mail}
	}

	// Add POSIX attributes if set
	if svcacct.IsPosix() {
		attrs["objectClass"] = append(attrs["objectClass"], "posixAccount")
		attrs["uidNumber"] = []string{strconv.Itoa(svcacct.GetUIDNumber())}
		attrs["gidNumber"] = []string{strconv.Itoa(svcacct.GetGIDNumber())}
		attrs["homeDirectory"] = []string{"/nonexistent"}
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
		c.config.LDAPEnforcer.PeopleBaseDN,
		c.config.LDAPEnforcer.SvcAcctBaseDN,
		c.config.LDAPEnforcer.GroupBaseDN,
		c.config.LDAPEnforcer.ManagedOU,
	)
	if err != nil {
		return nil, err
	}

	// Add all member DNs
	var memberDNs []string
	for _, member := range members {
		memberDNs = append(memberDNs, member.DN)
	}

	// If there are no members, add the bind DN to avoid empty group error
	if len(memberDNs) == 0 {
		memberDNs = append(memberDNs, c.config.LDAPEnforcer.BindDN)
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
	peopleOU := fmt.Sprintf("ou=%s,%s",
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.PeopleBaseDN)
	err := c.EnsureOUExists(peopleOU)
	if err != nil {
		return fmt.Errorf("failed to ensure people OU exists: %w", err)
	}

	// Ensure service account OU exists
	svcacctOU := fmt.Sprintf("ou=%s,%s",
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.SvcAcctBaseDN)
	err = c.EnsureOUExists(svcacctOU)
	if err != nil {
		return fmt.Errorf("failed to ensure service account OU exists: %w", err)
	}

	// Ensure group OU exists
	groupOU := fmt.Sprintf("ou=%s,%s",
		ldap.EscapeFilter(c.config.LDAPEnforcer.ManagedOU),
		c.config.LDAPEnforcer.GroupBaseDN)
	err = c.EnsureOUExists(groupOU)
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

	attrs := GetPersonAttributes(person)

	if exists {
		// Update existing person
		log.Printf("Updating person: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		// Create new person
		log.Printf("Creating person: %s", dn)
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

	attrs := GetSvcAcctAttributes(svcacct)

	if exists {
		// Update existing service account
		log.Printf("Updating service account: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		// Create new service account
		log.Printf("Creating service account: %s", dn)
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
		return err
	}

	if exists {
		// Update existing group
		log.Printf("Updating group: %s", dn)
		return c.ModifyEntry(dn, attrs, ldap.ReplaceAttribute)
	} else {
		// Create new group
		log.Printf("Creating group: %s", dn)
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
