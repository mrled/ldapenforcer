package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/pflag"
)

// Config represents the application configuration
type Config struct {
	// LDAPEnforcer configuration
	LDAPEnforcer LDAPEnforcerConfig `toml:"ldapenforcer"`

	// Processed includes (not in TOML)
	processedIncludes map[string]bool
}

// LDAPEnforcerConfig holds all the application settings
type LDAPEnforcerConfig struct {
	// LDAP URI (e.g. ldap://example.com:389)
	URI string `toml:"uri"`

	// DN for binding to LDAP
	BindDN string `toml:"bind_dn"`

	// Password for binding to LDAP
	Password string `toml:"password"`

	// File containing the password for binding to LDAP
	PasswordFile string `toml:"password_file"`

	// Base DN for users
	UserBaseDN string `toml:"user_base_dn"`

	// Base DN for services
	ServiceBaseDN string `toml:"service_base_dn"`

	// Base DN for groups
	GroupBaseDN string `toml:"group_base_dn"`

	// Name of the OU indicating managed objects
	ManagedOU string `toml:"managed_ou"`

	// List of config files to include
	Includes []string `toml:"includes"`
}

// LoadConfig loads configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	config := &Config{
		processedIncludes: make(map[string]bool),
	}

	// Load the main config file
	err := config.loadConfigFile(configFile)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// loadConfigFile loads a config file and processes includes
func (c *Config) loadConfigFile(configFile string) error {
	// Resolve the absolute path
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", configFile, err)
	}

	// Check if we've already processed this file
	if c.processedIncludes[absPath] {
		return nil
	}
	c.processedIncludes[absPath] = true

	// Read the config file
	var config Config
	_, err = toml.DecodeFile(absPath, &config)
	if err != nil {
		return fmt.Errorf("failed to decode config file %s: %w", absPath, err)
	}

	// Merge configs
	c.merge(&config)

	// Process includes
	configDir := filepath.Dir(absPath)
	for _, include := range config.LDAPEnforcer.Includes {
		var includePath string
		if filepath.IsAbs(include) {
			includePath = include
		} else {
			includePath = filepath.Join(configDir, include)
		}

		err := c.loadConfigFile(includePath)
		if err != nil {
			return fmt.Errorf("failed to load included config %s: %w", include, err)
		}
	}

	return nil
}

// merge merges another config into this one
func (c *Config) merge(other *Config) {
	// Only merge non-empty values
	if other.LDAPEnforcer.URI != "" {
		c.LDAPEnforcer.URI = other.LDAPEnforcer.URI
	}
	if other.LDAPEnforcer.BindDN != "" {
		c.LDAPEnforcer.BindDN = other.LDAPEnforcer.BindDN
	}
	if other.LDAPEnforcer.Password != "" {
		c.LDAPEnforcer.Password = other.LDAPEnforcer.Password
	}
	if other.LDAPEnforcer.PasswordFile != "" {
		c.LDAPEnforcer.PasswordFile = other.LDAPEnforcer.PasswordFile
	}
	if other.LDAPEnforcer.UserBaseDN != "" {
		c.LDAPEnforcer.UserBaseDN = other.LDAPEnforcer.UserBaseDN
	}
	if other.LDAPEnforcer.ServiceBaseDN != "" {
		c.LDAPEnforcer.ServiceBaseDN = other.LDAPEnforcer.ServiceBaseDN
	}
	if other.LDAPEnforcer.GroupBaseDN != "" {
		c.LDAPEnforcer.GroupBaseDN = other.LDAPEnforcer.GroupBaseDN
	}
	if other.LDAPEnforcer.ManagedOU != "" {
		c.LDAPEnforcer.ManagedOU = other.LDAPEnforcer.ManagedOU
	}
}

// GetPassword returns the LDAP password, loading it from the password file if specified
func (c *Config) GetPassword() (string, error) {
	// If password is directly specified, use it
	if c.LDAPEnforcer.Password != "" {
		return c.LDAPEnforcer.Password, nil
	}

	// Otherwise, try to load from the password file
	if c.LDAPEnforcer.PasswordFile != "" {
		data, err := os.ReadFile(c.LDAPEnforcer.PasswordFile)
		if err != nil {
			return "", fmt.Errorf("failed to read password file %s: %w", c.LDAPEnforcer.PasswordFile, err)
		}
		// Trim whitespace
		return strings.TrimSpace(string(data)), nil
	}

	return "", nil
}

// AddFlags adds the configuration flags to the provided flag set
func AddFlags(flags *pflag.FlagSet) {
	// config flag is now set in the root command directly
	flags.String("ldap-uri", "", "LDAP URI (e.g. ldap://example.com:389)")
	flags.String("bind-dn", "", "DN for binding to LDAP")
	flags.String("password", "", "Password for binding to LDAP")
	flags.String("password-file", "", "File containing the password for binding to LDAP")
	flags.String("user-base-dn", "", "Base DN for users")
	flags.String("service-base-dn", "", "Base DN for services")
	flags.String("group-base-dn", "", "Base DN for groups") 
	flags.String("managed-ou", "", "Name of the OU indicating managed objects")
}

// MergeWithFlags merges command line flag values into the config
func (c *Config) MergeWithFlags(flags *pflag.FlagSet) {
	if uri, _ := flags.GetString("ldap-uri"); uri != "" {
		c.LDAPEnforcer.URI = uri
	}
	if bindDN, _ := flags.GetString("bind-dn"); bindDN != "" {
		c.LDAPEnforcer.BindDN = bindDN
	}
	if password, _ := flags.GetString("password"); password != "" {
		c.LDAPEnforcer.Password = password
	}
	if passwordFile, _ := flags.GetString("password-file"); passwordFile != "" {
		c.LDAPEnforcer.PasswordFile = passwordFile
	}
	if userBaseDN, _ := flags.GetString("user-base-dn"); userBaseDN != "" {
		c.LDAPEnforcer.UserBaseDN = userBaseDN
	}
	if serviceBaseDN, _ := flags.GetString("service-base-dn"); serviceBaseDN != "" {
		c.LDAPEnforcer.ServiceBaseDN = serviceBaseDN
	}
	if groupBaseDN, _ := flags.GetString("group-base-dn"); groupBaseDN != "" {
		c.LDAPEnforcer.GroupBaseDN = groupBaseDN
	}
	if managedOU, _ := flags.GetString("managed-ou"); managedOU != "" {
		c.LDAPEnforcer.ManagedOU = managedOU
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.LDAPEnforcer.URI == "" {
		return fmt.Errorf("LDAP URI is required")
	}
	if c.LDAPEnforcer.BindDN == "" {
		return fmt.Errorf("bind DN is required")
	}
	
	// Check if either password or password file is provided
	if c.LDAPEnforcer.Password == "" && c.LDAPEnforcer.PasswordFile == "" {
		return fmt.Errorf("either password or password file must be provided")
	}
	
	if c.LDAPEnforcer.UserBaseDN == "" {
		return fmt.Errorf("user base DN is required")
	}
	if c.LDAPEnforcer.ServiceBaseDN == "" {
		return fmt.Errorf("service base DN is required")
	}
	if c.LDAPEnforcer.GroupBaseDN == "" {
		return fmt.Errorf("group base DN is required")
	}
	if c.LDAPEnforcer.ManagedOU == "" {
		return fmt.Errorf("managed OU is required")
	}
	
	return nil
}