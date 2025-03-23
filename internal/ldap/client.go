package ldap

import (
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/config"
)

// Client represents an LDAP client
type Client struct {
	conn *ldap.Conn
	config *config.Config
}

// NewClient creates a new LDAP client
func NewClient(cfg *config.Config) (*Client, error) {
	// Connect to LDAP server
	conn, err := ldap.DialURL(cfg.LDAPEnforcer.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Get the password from config or password file
	password, err := cfg.GetPassword()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get LDAP password: %w", err)
	}

	// Bind with DN and password
	err = conn.Bind(cfg.LDAPEnforcer.BindDN, password)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	return &Client{
		conn: conn,
		config: cfg,
	}, nil
}

// Close closes the LDAP connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// Search performs an LDAP search
func (c *Client) Search(baseDN, filter string, attributes []string) (*ldap.SearchResult, error) {
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, // No size limit
		0, // No time limit
		false,
		filter,
		attributes,
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	return result, nil
}

// CreateEntry creates a new LDAP entry
func (c *Client) CreateEntry(dn string, attributes map[string][]string) error {
	// Convert map to ldap.AddRequest
	addReq := ldap.NewAddRequest(dn, nil)
	for attr, values := range attributes {
		addReq.Attribute(attr, values)
	}

	// Execute add request
	err := c.conn.Add(addReq)
	if err != nil {
		return fmt.Errorf("failed to create LDAP entry: %w", err)
	}

	return nil
}

// ModifyEntry modifies an existing LDAP entry
func (c *Client) ModifyEntry(dn string, mods map[string][]string, operation int) error {
	// Convert map to ldap.ModifyRequest
	modReq := ldap.NewModifyRequest(dn, nil)
	for attr, values := range mods {
		switch operation {
		case ldap.AddAttribute:
			modReq.Add(attr, values)
		case ldap.ReplaceAttribute:
			modReq.Replace(attr, values)
		case ldap.DeleteAttribute:
			modReq.Delete(attr, values)
		default:
			return fmt.Errorf("invalid modify operation: %d", operation)
		}
	}

	// Execute modify request
	err := c.conn.Modify(modReq)
	if err != nil {
		return fmt.Errorf("failed to modify LDAP entry: %w", err)
	}

	return nil
}

// DeleteEntry deletes an LDAP entry
func (c *Client) DeleteEntry(dn string) error {
	delReq := ldap.NewDelRequest(dn, nil)
	err := c.conn.Del(delReq)
	if err != nil {
		return fmt.Errorf("failed to delete LDAP entry: %w", err)
	}

	return nil
}

// GetEntity retrieves an entity from LDAP
func (c *Client) GetEntity(dn string, attributes []string) (*ldap.Entry, error) {
	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, // No size limit
		0, // No time limit
		false,
		"(objectClass=*)",
		attributes,
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("no entry found for DN: %s", dn)
	}

	return result.Entries[0], nil
}

// EntryExists checks if an LDAP entry exists
func (c *Client) EntryExists(dn string) (bool, error) {
	searchRequest := ldap.NewSearchRequest(
		dn,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, // No size limit
		0, // No time limit
		false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		// If it's a "no such object" error, the entry doesn't exist
		if ldapErr, ok := err.(*ldap.Error); ok && ldapErr.ResultCode == ldap.LDAPResultNoSuchObject {
			return false, nil
		}
		return false, fmt.Errorf("LDAP search failed: %w", err)
	}

	return len(result.Entries) > 0, nil
}

// EnsureOUExists ensures that an OU exists, creating it if needed
func (c *Client) EnsureOUExists(dn string) error {
	exists, err := c.EntryExists(dn)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Create the OU
	attributes := map[string][]string{
		"objectClass": {"top", "organizationalUnit"},
		"ou":          {ldap.EscapeFilter(getOUFromDN(dn))},
	}

	log.Printf("Creating OU: %s", dn)
	return c.CreateEntry(dn, attributes)
}

// getOUFromDN extracts the OU name from a DN
func getOUFromDN(dn string) string {
	// This is a very simple implementation - a real one would be more robust
	// For a DN like "ou=name,dc=example,dc=com", it returns "name"
	entry, err := ldap.ParseDN(dn)
	if err != nil {
		return ""
	}

	// Only return the OU value if it's the first RDN
	if len(entry.RDNs) > 0 {
		for _, attr := range entry.RDNs[0].Attributes {
			if attr.Type == "ou" {
				return attr.Value
			}
		}
	}

	return ""
}