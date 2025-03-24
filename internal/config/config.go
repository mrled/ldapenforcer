package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/mrled/ldapenforcer/internal/model"
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

	// Command to execute to retrieve the password
	PasswordCommand string `toml:"password_command"`

	// Execute password command via shell (using sh -c)
	PasswordCommandViaShell bool `toml:"password_command_via_shell"`

	// Path to CA certificate file for LDAPS
	CACertFile string `toml:"ca_cert_file"`

	// Logging configuration
	Logging LoggingConfig `toml:"logging"`

	// Base DN for people
	PeopleBaseDN string `toml:"people_base_dn"`

	// Base DN for service accounts
	SvcAcctBaseDN string `toml:"svcacct_base_dn"`

	// Base DN for groups
	GroupBaseDN string `toml:"group_base_dn"`

	// Name of the OU indicating managed objects
	ManagedOU string `toml:"managed_ou"`

	// List of config files to include
	Includes []string `toml:"includes"`

	// Person configurations - map of uid to person config
	Person map[string]*model.Person `toml:"person"`

	// Service account configurations - map of uid to service account config
	SvcAcct map[string]*model.SvcAcct `toml:"svcacct"`

	// Group configurations - map of group name to group config
	Group map[string]*model.Group `toml:"group"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	// Level for main application logs
	Level string `toml:"level"`

	// LDAP specific logging configuration
	LDAP LDAPLoggingConfig `toml:"ldap"`
}

// LDAPLoggingConfig holds LDAP-specific logging configuration
type LDAPLoggingConfig struct {
	// Level for LDAP-related logs
	Level string `toml:"level"`
}

// LoadConfig loads configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	config := &Config{
		processedIncludes: make(map[string]bool),
	}

	// Initialize the config structure to avoid nil pointers
	config.LDAPEnforcer.Person = make(map[string]*model.Person)
	config.LDAPEnforcer.SvcAcct = make(map[string]*model.SvcAcct)
	config.LDAPEnforcer.Group = make(map[string]*model.Group)
	config.LDAPEnforcer.Includes = make([]string, 0)

	// Store the directory of the main config file
	absConfigFile, err := filepath.Abs(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for config file: %w", err)
	}
	configDir = filepath.Dir(absConfigFile)

	// Load the main config file
	err = config.loadConfigFile(configFile)
	if err != nil {
		return nil, err
	}

	// Load configuration from environment variables, overriding settings from the config file
	config.MergeWithEnv()

	// Set defaults for the logging configuration
	if config.LDAPEnforcer.Logging.Level == "" {
		config.LDAPEnforcer.Logging.Level = "ERROR"
	}

	if config.LDAPEnforcer.Logging.LDAP.Level == "" {
		config.LDAPEnforcer.Logging.LDAP.Level = config.LDAPEnforcer.Logging.Level
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

	// Store the includes to process after merging
	includes := make([]string, len(config.LDAPEnforcer.Includes))
	copy(includes, config.LDAPEnforcer.Includes)

	// Get the directory for this config file
	configDir := filepath.Dir(absPath)

	// First merge the current config file into our config
	c.merge(&config)

	// Process includes - process them AFTER merging the current file
	// This ensures that included files can override settings from the parent file
	for _, include := range includes {
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
	if other.LDAPEnforcer.PasswordCommand != "" {
		c.LDAPEnforcer.PasswordCommand = other.LDAPEnforcer.PasswordCommand
	}
	// For boolean flags like PasswordCommandViaShell, only merge if true
	if other.LDAPEnforcer.PasswordCommandViaShell {
		c.LDAPEnforcer.PasswordCommandViaShell = true
	}
	if other.LDAPEnforcer.CACertFile != "" {
		c.LDAPEnforcer.CACertFile = other.LDAPEnforcer.CACertFile
	}
	// Handle the logging structure
	if other.LDAPEnforcer.Logging.Level != "" {
		c.LDAPEnforcer.Logging.Level = other.LDAPEnforcer.Logging.Level
	}
	if other.LDAPEnforcer.Logging.LDAP.Level != "" {
		c.LDAPEnforcer.Logging.LDAP.Level = other.LDAPEnforcer.Logging.LDAP.Level
	}
	if other.LDAPEnforcer.PeopleBaseDN != "" {
		c.LDAPEnforcer.PeopleBaseDN = other.LDAPEnforcer.PeopleBaseDN
	}
	if other.LDAPEnforcer.SvcAcctBaseDN != "" {
		c.LDAPEnforcer.SvcAcctBaseDN = other.LDAPEnforcer.SvcAcctBaseDN
	}
	if other.LDAPEnforcer.GroupBaseDN != "" {
		c.LDAPEnforcer.GroupBaseDN = other.LDAPEnforcer.GroupBaseDN
	}
	if other.LDAPEnforcer.ManagedOU != "" {
		c.LDAPEnforcer.ManagedOU = other.LDAPEnforcer.ManagedOU
	}

	// Make sure we also append any includes
	c.LDAPEnforcer.Includes = append(c.LDAPEnforcer.Includes, other.LDAPEnforcer.Includes...)

	// Merge people
	if other.LDAPEnforcer.Person != nil {
		if c.LDAPEnforcer.Person == nil {
			c.LDAPEnforcer.Person = make(map[string]*model.Person)
		}
		for uid, person := range other.LDAPEnforcer.Person {
			// Set the Username field with the uid (map key)
			person.Username = uid
			c.LDAPEnforcer.Person[uid] = person
		}
	}

	// Merge service accounts
	if other.LDAPEnforcer.SvcAcct != nil {
		if c.LDAPEnforcer.SvcAcct == nil {
			c.LDAPEnforcer.SvcAcct = make(map[string]*model.SvcAcct)
		}
		for uid, svcacct := range other.LDAPEnforcer.SvcAcct {
			// Set the Username field with the uid (map key)
			svcacct.Username = uid
			c.LDAPEnforcer.SvcAcct[uid] = svcacct
		}
	}

	// Merge groups
	if other.LDAPEnforcer.Group != nil {
		if c.LDAPEnforcer.Group == nil {
			c.LDAPEnforcer.Group = make(map[string]*model.Group)
		}
		for groupname, group := range other.LDAPEnforcer.Group {
			c.LDAPEnforcer.Group[groupname] = group
		}
	}
}

// configDir stores the directory of the main config file
var configDir string

// GetConfigDir returns the directory of the main config file
func GetConfigDir() (string, error) {
	if configDir == "" {
		return "", fmt.Errorf("config directory not set, config file may not have been loaded")
	}
	return configDir, nil
}

// GetPassword returns the LDAP password, loading it from the password file or command if specified
func (c *Config) GetPassword() (string, error) {
	// If password is directly specified, use it
	if c.LDAPEnforcer.Password != "" {
		return c.LDAPEnforcer.Password, nil
	}

	// Try to load from the password file
	if c.LDAPEnforcer.PasswordFile != "" {
		// Resolve password file path relative to config file if it's not absolute
		passwordFilePath := c.LDAPEnforcer.PasswordFile
		if !filepath.IsAbs(passwordFilePath) && configDir != "" {
			passwordFilePath = filepath.Join(configDir, passwordFilePath)
		}

		data, err := os.ReadFile(passwordFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read password file %s: %w", passwordFilePath, err)
		}
		// Trim whitespace
		return strings.TrimSpace(string(data)), nil
	}

	// Try to execute the password command
	if c.LDAPEnforcer.PasswordCommand != "" {
		logging.DefaultLogger.Debug("Executing password command to retrieve LDAP credentials")

		// Use shell if explicitly requested via password_command_via_shell
		if c.LDAPEnforcer.PasswordCommandViaShell {
			logging.DefaultLogger.Debug("Executing password command via shell (sh -c)")
			// Use shell to execute the command
			cmd := exec.Command("sh", "-c", c.LDAPEnforcer.PasswordCommand)

			// Capture stdout
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = os.Stderr

			// Run the command
			err := cmd.Run()
			if err != nil {
				logging.DefaultLogger.Error("Error executing shell password command: %v", err)
				return "", fmt.Errorf("failed to execute shell password command: %w", err)
			}

			// Return the password from stdout, trimming whitespace
			result := strings.TrimSpace(stdout.String())
			logging.DefaultLogger.Debug("Successfully retrieved password from command (length: %d)", len(result))
			return result, nil
		} else {
			// Split the command and its arguments for direct execution
			parts, err := parseCommandString(c.LDAPEnforcer.PasswordCommand)
			if err != nil {
				return "", fmt.Errorf("failed to parse password command: %w", err)
			}

			if len(parts) == 0 {
				return "", fmt.Errorf("empty password command")
			}

			logging.DefaultLogger.Debug("Executing direct command: %s", parts[0])
			// Create the command
			cmd := exec.Command(parts[0], parts[1:]...)

			// Capture stdout
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = os.Stderr

			// Run the command
			err = cmd.Run()
			if err != nil {
				logging.DefaultLogger.Error("Error executing password command: %v", err)
				return "", fmt.Errorf("failed to execute password command: %w", err)
			}

			// Return the password from stdout, trimming whitespace
			result := strings.TrimSpace(stdout.String())
			logging.DefaultLogger.Debug("Successfully retrieved password from command (length: %d)", len(result))
			return result, nil
		}
	}

	return "", nil
}

// MergeWithEnv loads configuration values from environment variables
func (c *Config) MergeWithEnv() {
	// Core LDAP connection settings
	if val := os.Getenv("LDAPENFORCER_URI"); val != "" {
		c.LDAPEnforcer.URI = val
	}
	if val := os.Getenv("LDAPENFORCER_BIND_DN"); val != "" {
		c.LDAPEnforcer.BindDN = val
	}
	if val := os.Getenv("LDAPENFORCER_PASSWORD"); val != "" {
		c.LDAPEnforcer.Password = val
	}
	if val := os.Getenv("LDAPENFORCER_PASSWORD_FILE"); val != "" {
		c.LDAPEnforcer.PasswordFile = val
	}
	if val := os.Getenv("LDAPENFORCER_PASSWORD_COMMAND"); val != "" {
		c.LDAPEnforcer.PasswordCommand = val
	}
	if val := os.Getenv("LDAPENFORCER_PASSWORD_COMMAND_VIA_SHELL"); val != "" {
		boolValue, err := strconv.ParseBool(val)
		if err == nil && boolValue {
			c.LDAPEnforcer.PasswordCommandViaShell = true
		}
	}
	if val := os.Getenv("LDAPENFORCER_CA_CERT_FILE"); val != "" {
		c.LDAPEnforcer.CACertFile = val
	}

	// Logging configuration
	if val := os.Getenv("LDAPENFORCER_LOG_LEVEL"); val != "" {
		c.LDAPEnforcer.Logging.Level = val
	}
	if val := os.Getenv("LDAPENFORCER_LDAP_LOG_LEVEL"); val != "" {
		c.LDAPEnforcer.Logging.LDAP.Level = val
	}

	// Directory structure
	if val := os.Getenv("LDAPENFORCER_PEOPLE_BASE_DN"); val != "" {
		c.LDAPEnforcer.PeopleBaseDN = val
	}
	if val := os.Getenv("LDAPENFORCER_SVCACCT_BASE_DN"); val != "" {
		c.LDAPEnforcer.SvcAcctBaseDN = val
	}
	if val := os.Getenv("LDAPENFORCER_GROUP_BASE_DN"); val != "" {
		c.LDAPEnforcer.GroupBaseDN = val
	}
	if val := os.Getenv("LDAPENFORCER_MANAGED_OU"); val != "" {
		c.LDAPEnforcer.ManagedOU = val
	}

	// Includes - process as comma-separated list
	if val := os.Getenv("LDAPENFORCER_INCLUDES"); val != "" {
		includes := strings.Split(val, ",")
		for i := range includes {
			includes[i] = strings.TrimSpace(includes[i])
		}
		c.LDAPEnforcer.Includes = append(c.LDAPEnforcer.Includes, includes...)
	}
}

// AddFlags adds the configuration flags to the provided flag set
func AddFlags(flags *pflag.FlagSet) {
	// config flag is now set in the root command directly
	flags.String("ldap-uri", "", "LDAP URI (e.g. ldap://example.com:389)")
	flags.String("bind-dn", "", "DN for binding to LDAP")
	flags.String("password", "", "Password for binding to LDAP")
	flags.String("password-file", "", "File containing the password for binding to LDAP")
	flags.String("password-command", "", "Command to execute to retrieve the password")
	flags.Bool("password-command-via-shell", false, "Execute password command via shell (using sh -c)")
	flags.String("ca-cert-file", "", "Path to CA certificate file for LDAPS")
	flags.String("log-level", "ERROR", "Main log level (ERROR, WARN, INFO, DEBUG, TRACE)")
	flags.String("ldap-log-level", "ERROR", "LDAP-specific log level (ERROR, WARN, INFO, DEBUG, TRACE)")
	flags.String("people-base-dn", "", "Base DN for people")
	flags.String("svcacct-base-dn", "", "Base DN for service accounts")
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
	if passwordCommand, _ := flags.GetString("password-command"); passwordCommand != "" {
		c.LDAPEnforcer.PasswordCommand = passwordCommand
	}
	if viaShell, _ := flags.GetBool("password-command-via-shell"); viaShell {
		c.LDAPEnforcer.PasswordCommandViaShell = true
	}
	if caCertFile, _ := flags.GetString("ca-cert-file"); caCertFile != "" {
		c.LDAPEnforcer.CACertFile = caCertFile
	}
	if logLevel, _ := flags.GetString("log-level"); logLevel != "" {
		c.LDAPEnforcer.Logging.Level = logLevel
	}
	if ldapLogLevel, _ := flags.GetString("ldap-log-level"); ldapLogLevel != "" {
		c.LDAPEnforcer.Logging.LDAP.Level = ldapLogLevel
	}
	if peopleBaseDN, _ := flags.GetString("people-base-dn"); peopleBaseDN != "" {
		c.LDAPEnforcer.PeopleBaseDN = peopleBaseDN
	}
	if svcAcctBaseDN, _ := flags.GetString("svcacct-base-dn"); svcAcctBaseDN != "" {
		c.LDAPEnforcer.SvcAcctBaseDN = svcAcctBaseDN
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

	// Check if either password, password file, or password command is provided
	if c.LDAPEnforcer.Password == "" &&
		c.LDAPEnforcer.PasswordFile == "" &&
		c.LDAPEnforcer.PasswordCommand == "" {
		return fmt.Errorf("one of password, password_file, or password_command must be provided")
	}

	if c.LDAPEnforcer.PeopleBaseDN == "" {
		return fmt.Errorf("people base DN is required")
	}
	if c.LDAPEnforcer.SvcAcctBaseDN == "" {
		return fmt.Errorf("service account base DN is required")
	}
	if c.LDAPEnforcer.GroupBaseDN == "" {
		return fmt.Errorf("group base DN is required")
	}
	if c.LDAPEnforcer.ManagedOU == "" {
		return fmt.Errorf("managed OU is required")
	}

	return nil
}

// parseCommandString parses a command string into command and arguments
// This handles quoted arguments correctly
func parseCommandString(command string) ([]string, error) {
	var parts []string
	var current string
	var inQuotes bool
	var quoteChar rune

	for _, char := range command {
		switch {
		case char == '"' || char == '\'':
			// Toggle quotes
			if inQuotes && char == quoteChar {
				inQuotes = false
			} else if !inQuotes {
				inQuotes = true
				quoteChar = char
			} else {
				// Add the quote character if we're inside a different type of quotes
				current += string(char)
			}
		case char == ' ' && !inQuotes:
			// End of part
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			// Add to current part
			current += string(char)
		}
	}

	// Add the last part if there is one
	if current != "" {
		parts = append(parts, current)
	}

	// If we're still in quotes, that's an error
	if inQuotes {
		return nil, fmt.Errorf("unclosed quotes in command string")
	}

	return parts, nil
}
