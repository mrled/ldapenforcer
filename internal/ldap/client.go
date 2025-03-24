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
)

// Client represents an LDAP client
type Client struct {
	conn   *ldap.Conn
	config *config.Config
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
		conn, err = ldap.DialURL(cfg.LDAPEnforcer.URI, ldap.DialWithTLSConfig(tlsConfig))
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
			conn.Close()
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
			conn.Close()
		}
		return nil, fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	logging.LDAPProtocolLogger.Info("Successfully authenticated to LDAP server")

	return &Client{
		conn:   conn,
		config: cfg,
	}, nil
}

// Close closes the LDAP connection
func (c *Client) Close() {
	if c != nil && c.conn != nil {
		c.conn.Close()
	}
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
	err := c.conn.Modify(modReq)
	if err != nil {
		logging.LDAPProtocolLogger.Error("Failed to modify LDAP entry: %v", err)
		return fmt.Errorf("failed to modify LDAP entry: %w", err)
	}

	logging.LDAPProtocolLogger.Info("Successfully modified LDAP entry: %s", dn)
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
