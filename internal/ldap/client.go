package ldap

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/mrled/ldapenforcer/internal/model"
)

// LDAPClientInterface defines the interface that both real and mock LDAP clients implement
type LDAPClientInterface interface {
	// Connection management
	Close() error

	// DN generation
	PersonToDN(uid string) string
	SvcAcctToDN(uid string) string
	GroupToDN(groupname string) string

	// Basic LDAP operations
	EntryExists(dn string) (bool, error)
	CreateEntry(dn string, attrs map[string][]string) error
	ModifyEntry(dn string, attrs map[string][]string, modType int) error
	DeleteEntry(dn string) error
	GetExistingEntries(ou string, entryType string) (map[string]string, error)

	// OU management
	EnsureManagedOUsExist() error
	EnsureOUExists(ou string) error

	// Sync operations
	SyncPerson(uid string, person *model.Person) error
	SyncSvcAcct(uid string, svcacct *model.SvcAcct) error
	SyncGroup(groupname string, group *model.Group) error
	SyncAll() error

	// Dependency resolution for internal implementation
	getGroupDependencies(groupname string, processedGroups map[string]bool) ([]string, []string)
	topologicalSortGroups(deps map[string][]string) []string
}

// BaseClient implements shared functionality across real and mock clients
type BaseClient struct {
	config *config.Config
}

// PersonToDN converts a person UID to a DN
func (b *BaseClient) PersonToDN(uid string) string {
	return fmt.Sprintf("uid=%s,%s",
		ldap.EscapeFilter(uid),
		b.config.LDAPEnforcer.EnforcedPeopleOU)
}

// SvcAcctToDN converts a service account UID to a DN
func (b *BaseClient) SvcAcctToDN(uid string) string {
	return fmt.Sprintf("uid=%s,%s",
		ldap.EscapeFilter(uid),
		b.config.LDAPEnforcer.EnforcedSvcAcctOU)
}

// GroupToDN converts a group name to a DN
func (b *BaseClient) GroupToDN(groupname string) string {
	return fmt.Sprintf("cn=%s,%s",
		ldap.EscapeFilter(groupname),
		b.config.LDAPEnforcer.EnforcedGroupOU)
}

// getGroupDependencies returns a list of groups that this group depends on
// and a list of unresolvable member UIDs
func (b *BaseClient) getGroupDependencies(groupname string, processedGroups map[string]bool) ([]string, []string) {
	// Mark this group as visited to avoid infinite recursion
	processedGroups[groupname] = true

	// Get the group
	group, ok := b.config.LDAPEnforcer.Group[groupname]
	if !ok {
		return nil, nil
	}

	// Track dependencies and unresolvable members
	var dependencies []string
	var unresolvableMembers []string

	// Check people members
	for _, uid := range group.People {
		if _, ok := b.config.LDAPEnforcer.Person[uid]; !ok {
			unresolvableMembers = append(unresolvableMembers, uid)
		}
	}

	// Check service account members
	for _, uid := range group.SvcAccts {
		if _, ok := b.config.LDAPEnforcer.SvcAcct[uid]; !ok {
			unresolvableMembers = append(unresolvableMembers, uid)
		}
	}

	// Check group members and recursively build dependencies
	for _, nestedGroupName := range group.Groups {
		// Add as dependency
		dependencies = append(dependencies, nestedGroupName)

		// If we haven't processed this nested group yet, get its dependencies too
		if !processedGroups[nestedGroupName] {
			nestedDeps, nestedUnres := b.getGroupDependencies(nestedGroupName, processedGroups)
			dependencies = append(dependencies, nestedDeps...)
			unresolvableMembers = append(unresolvableMembers, nestedUnres...)
		}
	}

	return dependencies, unresolvableMembers
}

// topologicalSortGroups returns a slice of group names in topological order
func (b *BaseClient) topologicalSortGroups(deps map[string][]string) []string {
	// First, get all groups (including those with no dependencies)
	allGroups := make(map[string]bool)
	for groupname := range b.config.LDAPEnforcer.Group {
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
			logging.DefaultLogger.Warn("Cyclic dependency detected involving group: %s", node)
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

// Client represents a real LDAP client that connects to an LDAP server
type Client struct {
	BaseClient
	conn *ldap.Conn
}

// NewClient creates a new LDAP client
func NewClient(cfg *config.Config) (*Client, error) {
	var conn *ldap.Conn
	var err error

	// Check if using LDAPS and configure TLS if needed
	isLDAPS := strings.HasPrefix(strings.ToLower(cfg.LDAPEnforcer.URI), "ldaps://")

	logging.LDAPProtocolLogger.Debug("Connecting to LDAP server at %s", cfg.LDAPEnforcer.URI)
	if isLDAPS {
		logging.LDAPProtocolLogger.Debug("Using LDAPS (secure) connection")
	}

	if isLDAPS && cfg.LDAPEnforcer.CACertFile != "" {
		// Get the CA certificate file path, resolving it if needed
		caCertPath := cfg.LDAPEnforcer.CACertFile

		// If it's a relative path, make it relative to the config file directory
		if !filepath.IsAbs(caCertPath) {
			configDir, err := getConfigDir()
			if err == nil && configDir != "" {
				caCertPath = filepath.Join(configDir, caCertPath)
			}
		}

		logging.LDAPProtocolLogger.Debug("Using CA certificate from %s", caCertPath)

		// Load CA certificate
		tlsConfig, err := createTLSConfig(caCertPath)
		if err != nil {
			logging.LDAPProtocolLogger.Error("Failed to create TLS config: %v", err)
			return nil, fmt.Errorf("failed to create TLS config: %w", err)
		}

		// Connect with TLS
		logging.LDAPProtocolLogger.Trace("Using custom TLS config with CA certificate from %s", caCertPath)
		// Using a separate error variable to avoid shadowing
		var dialErr error
		conn, dialErr = ldap.DialURL(cfg.LDAPEnforcer.URI, ldap.DialWithTLSConfig(tlsConfig))
		if dialErr != nil {
			logging.LDAPProtocolLogger.Error("Failed to connect to LDAP server with TLS: %v", dialErr)
			return nil, fmt.Errorf("failed to connect to LDAP server with TLS: %w", dialErr)
		}
	} else {
		// Connect without TLS or with default TLS
		logging.LDAPProtocolLogger.Debug("Dialing LDAP URL %s", cfg.LDAPEnforcer.URI)
		conn, err = ldap.DialURL(cfg.LDAPEnforcer.URI)
	}

	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to connect to LDAP server: %v", err)
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	logging.LDAPProtocolLogger.Debug("Successfully connected to LDAP server")

	// Get the password from config, password file, or password command
	logging.LDAPProtocolLogger.Debug("Getting LDAP bind password")
	password, err := cfg.GetPassword()
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to get LDAP password: %v", err)
		// Only close the connection if it was successfully established
		if conn != nil {
			logging.LDAPProtocolLogger.Debug("Closing LDAP connection due to password retrieval failure")
			if err := conn.Close(); err != nil {
				logging.LDAPProtocolLogger.Warn("Error closing LDAP connection: %v", err)
			}
		}
		return nil, fmt.Errorf("failed to get LDAP password: %w", err)
	}

	// Bind with DN and password
	logging.LDAPProtocolLogger.Debug("Binding to LDAP server with DN: %s", cfg.LDAPEnforcer.BindDN)
	err = conn.Bind(cfg.LDAPEnforcer.BindDN, password)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to bind to LDAP server: %v", err)
		// Only close the connection if it was successfully established
		if conn != nil {
			logging.LDAPProtocolLogger.Debug("Closing LDAP connection due to bind failure")
			if err := conn.Close(); err != nil {
				logging.LDAPProtocolLogger.Warn("Error closing LDAP connection: %v", err)
			}
		}
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	logging.LDAPProtocolLogger.Trace("Successfully authenticated to LDAP server")

	return &Client{
		BaseClient: BaseClient{
			config: cfg,
		},
		conn: conn,
	}, nil
}

// Close closes the LDAP connection
func (c *Client) Close() error {
	if c != nil && c.conn != nil {
		if err := c.conn.Close(); err != nil {
			logging.LDAPProtocolLogger.Warn("Error closing LDAP connection: %v", err)
			return fmt.Errorf("error closing LDAP connection: %w", err)
		}
		logging.LDAPProtocolLogger.Debug("LDAP connection closed successfully")
	}
	return nil
}

// Search performs an LDAP search
func (c *Client) Search(baseDN, filter string, attributes []string) (*ldap.SearchResult, error) {
	logging.LDAPProtocolLogger.Trace("LDAP Search details - Base DN: %s, Filter: %s, Attributes: %v", baseDN, filter, attributes)

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
		logging.LDAPProtocolLogger.Error("LDAP search failed: %v", err)
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	logging.LDAPProtocolLogger.Trace("LDAP Search complete (found %d entries): %+v", len(result.Entries), result)

	return result, nil
}

// CreateEntry creates a new LDAP entry
func (c *Client) CreateEntry(dn string, attributes map[string][]string) error {
	logging.LDAPProtocolLogger.Trace("Creating LDAP entry: %+v", attributes)

	// Convert map to ldap.AddRequest
	addReq := ldap.NewAddRequest(dn, nil)
	for attr, values := range attributes {
		addReq.Attribute(attr, values)
	}

	// Execute add request
	err := c.conn.Add(addReq)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to create LDAP entry: %v", err)
		return fmt.Errorf("failed to create LDAP entry: %w", err)
	}

	logging.LDAPProtocolLogger.Info("Successfully created LDAP entry: %s", dn)
	return nil
}

// ModifyEntry modifies an existing LDAP entry
func (c *Client) ModifyEntry(dn string, mods map[string][]string, operation int) error {
	// Get operation name for logging
	logging.LDAPProtocolLogger.Trace("Modifying LDAP entry: %+v", mods)

	// Convert map to ldap.ModifyRequest
	modReq := ldap.NewModifyRequest(dn, nil)
	for attr, values := range mods {
		switch operation {
		case ldap.AddAttribute:
			logging.LDAPProtocolLogger.Trace("Adding attribute %s with values: %v", attr, values)
			modReq.Add(attr, values)
		case ldap.ReplaceAttribute:
			logging.LDAPProtocolLogger.Trace("Replacing attribute %s with values: %v", attr, values)
			modReq.Replace(attr, values)
		case ldap.DeleteAttribute:
			logging.LDAPProtocolLogger.Trace("Deleting attribute %s with values: %v", attr, values)
			modReq.Delete(attr, values)
		default:
			logging.LDAPProtocolLogger.Error("Invalid LDAP modify operation: %d", operation)
			return fmt.Errorf("invalid modify operation: %d", operation)
		}
	}

	// Execute modify request
	logging.LDAPProtocolLogger.Trace("Sending LDAP Modify request: %+v", modReq)
	err := c.conn.Modify(modReq)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to modify LDAP entry: %v", err)
		return fmt.Errorf("failed to modify LDAP entry: %w", err)
	}

	logging.LDAPProtocolLogger.Trace("Successfully modified LDAP entry: %s", dn)
	return nil
}

// DeleteEntry deletes an LDAP entry
func (c *Client) DeleteEntry(dn string) error {
	logging.LDAPProtocolLogger.Trace("Sending LDAP delete operation for DN=%s", dn)

	delReq := ldap.NewDelRequest(dn, nil)
	err := c.conn.Del(delReq)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to delete LDAP entry: %v", err)
		return fmt.Errorf("failed to delete LDAP entry: %w", err)
	}

	logging.LDAPProtocolLogger.Info("Successfully deleted LDAP entry: %s", dn)
	return nil
}

// GetEntity retrieves an entity from LDAP
func (c *Client) GetEntity(dn string, attributes []string) (*ldap.Entry, error) {
	logging.LDAPProtocolLogger.Trace("LDAP entity fetch - DN: %s, Attributes: %v", dn, attributes)

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
		logging.LDAPProtocolLogger.Error("LDAP get entity failed: %v", err)
		return nil, fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(result.Entries) == 0 {
		logging.LDAPProtocolLogger.Debug("No LDAP entry found for DN: %s", dn)
		return nil, fmt.Errorf("no entry found for DN: %s", dn)
	}

	logging.LDAPProtocolLogger.Trace("Successfully retrieved LDAP entity: %+v", result.Entries[0])
	return result.Entries[0], nil
}

// EntryExists checks if an LDAP entry exists
func (c *Client) EntryExists(dn string) (bool, error) {
	logging.LDAPProtocolLogger.Trace("Checking if LDAP entry exists: DN=%s", dn)

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
			logging.LDAPProtocolLogger.Debug("LDAP entry does not exist: %s", dn)
			return false, nil
		}
		logging.LDAPProtocolLogger.Error("LDAP existence check failed: %v", err)
		return false, fmt.Errorf("LDAP search failed: %w", err)
	}

	exists := len(result.Entries) > 0
	logging.LDAPProtocolLogger.Debug("LDAP entry exists check for %s: %v", dn, exists)
	return exists, nil
}

// EnsureOUExists ensures that an OU exists, creating it if needed
func (c *Client) EnsureOUExists(dn string) error {
	logging.LDAPProtocolLogger.Debug("Ensuring OU exists: %s", dn)

	exists, err := c.EntryExists(dn)
	if err != nil {
		return err
	}

	if exists {
		logging.LDAPProtocolLogger.Debug("OU already exists: %s", dn)
		return nil
	}

	// Create the OU
	ouName := getOUFromDN(dn)
	logging.LDAPProtocolLogger.Info("Creating OU '%s' with DN: %s", ouName, dn)

	attributes := map[string][]string{
		"objectClass": {"top", "organizationalUnit"},
		"ou":          {ldap.EscapeFilter(ouName)},
	}

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

// createTLSConfig creates a TLS configuration for LDAPS connections
func createTLSConfig(caCertFile string) (*tls.Config, error) {
	logging.LDAPProtocolLogger.Debug("Creating TLS config with CA certificate: %s", caCertFile)

	// Create a certificate pool with system CA certificates
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		logging.LDAPProtocolLogger.Debug("System cert pool not available, creating a new one")
		// If system cert pool is not available, create a new one
		rootCAs = x509.NewCertPool()
	}

	// Read CA certificate
	logging.LDAPProtocolLogger.Trace("Reading CA certificate from %s", caCertFile)
	caCert, err := os.ReadFile(caCertFile)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to read CA certificate file: %v", err)
		return nil, fmt.Errorf("failed to read CA certificate file %s: %w", caCertFile, err)
	}

	// Add CA certificate to the pool
	if !rootCAs.AppendCertsFromPEM(caCert) {
		logging.LDAPProtocolLogger.Error("Failed to append CA certificate from %s", caCertFile)
		return nil, fmt.Errorf("failed to append CA certificate from %s", caCertFile)
	}

	logging.LDAPProtocolLogger.Trace("Successfully added CA certificate to trust store")

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs: rootCAs,
	}

	return tlsConfig, nil
}

// getConfigDir returns the directory of the main config file
// by accessing the exported function from the config package
func getConfigDir() (string, error) {
	return config.GetConfigDir()
}
