package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	// Main log level (ERROR, WARN, INFO, DEBUG, TRACE)
	MainLogLevel string `toml:"main_log_level"`

	// LDAP-specific log level (ERROR, WARN, INFO, DEBUG, TRACE)
	LDAPLogLevel string `toml:"ldap_log_level"`

	// Full OU for enforced people
	EnforcedPeopleOU string `toml:"enforced_people_ou"`

	// Full OU for enforced service accounts
	EnforcedSvcAcctOU string `toml:"enforced_svcacct_ou"`

	// Full OU for enforced groups
	EnforcedGroupOU string `toml:"enforced_group_ou"`

	// Interval for polling config file changes (when poll is enabled via command line)
	PollConfigInterval string `toml:"poll_config_interval"`

	// Interval for polling LDAP server for changes (when poll is enabled via command line)
	PollLDAPInterval string `toml:"poll_ldap_interval"`

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

	// Store the main config file path for monitoring
	SetMainConfigFile(absConfigFile)

	// Set defaults for the logging configuration (lowest precedence)
	config.LDAPEnforcer.MainLogLevel = "INFO"
	config.LDAPEnforcer.LDAPLogLevel = "INFO"
	config.LDAPEnforcer.PollConfigInterval = "10s"
	config.LDAPEnforcer.PollLDAPInterval = "24h"

	// Load the main config file (second lowest precedence)
	err = config.loadConfigFile(configFile)
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
	if other.LDAPEnforcer.MainLogLevel != "" {
		c.LDAPEnforcer.MainLogLevel = other.LDAPEnforcer.MainLogLevel
	}
	if other.LDAPEnforcer.LDAPLogLevel != "" {
		c.LDAPEnforcer.LDAPLogLevel = other.LDAPEnforcer.LDAPLogLevel
	}
	if other.LDAPEnforcer.EnforcedPeopleOU != "" {
		c.LDAPEnforcer.EnforcedPeopleOU = other.LDAPEnforcer.EnforcedPeopleOU
	}
	if other.LDAPEnforcer.EnforcedSvcAcctOU != "" {
		c.LDAPEnforcer.EnforcedSvcAcctOU = other.LDAPEnforcer.EnforcedSvcAcctOU
	}
	if other.LDAPEnforcer.EnforcedGroupOU != "" {
		c.LDAPEnforcer.EnforcedGroupOU = other.LDAPEnforcer.EnforcedGroupOU
	}
	if other.LDAPEnforcer.PollConfigInterval != "" {
		c.LDAPEnforcer.PollConfigInterval = other.LDAPEnforcer.PollConfigInterval
	}
	if other.LDAPEnforcer.PollLDAPInterval != "" {
		c.LDAPEnforcer.PollLDAPInterval = other.LDAPEnforcer.PollLDAPInterval
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
var mainConfigFile string
var configFileModTimes map[string]time.Time

// GetConfigDir returns the directory of the main config file
func GetConfigDir() (string, error) {
	if configDir == "" {
		return "", fmt.Errorf("config directory not set, config file may not have been loaded")
	}
	return configDir, nil
}

// GetMainConfigFile returns the path to the main config file
func GetMainConfigFile() string {
	return mainConfigFile
}

// SetMainConfigFile sets the path to the main config file
func SetMainConfigFile(filepath string) {
	mainConfigFile = filepath
}

// InitConfigFileMonitoring initializes the config file modification time map
func InitConfigFileMonitoring(cfg *Config) error {
	configFileModTimes = make(map[string]time.Time)

	// Add main config file
	if mainConfigFile == "" {
		return fmt.Errorf("main config file path is not set")
	}

	abspath, err := filepath.Abs(mainConfigFile)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for %s: %w", mainConfigFile, err)
	}

	info, err := os.Stat(abspath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", abspath, err)
	}

	configFileModTimes[abspath] = info.ModTime()

	// Add all included config files
	return addIncludedConfigFiles(cfg, abspath)
}

// addIncludedConfigFiles recursively adds all included config files to the monitoring map
func addIncludedConfigFiles(cfg *Config, parentFile string) error {
	// Get the directory of the parent file
	parentDir := filepath.Dir(parentFile)

	// Process all includes
	for _, include := range cfg.LDAPEnforcer.Includes {
		var includePath string
		if filepath.IsAbs(include) {
			includePath = include
		} else {
			includePath = filepath.Join(parentDir, include)
		}

		// Resolve absolute path
		absIncludePath, err := filepath.Abs(includePath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for %s: %w", includePath, err)
		}

		// Skip if already processed
		if _, exists := configFileModTimes[absIncludePath]; exists {
			continue
		}

		// Get file info
		info, err := os.Stat(absIncludePath)
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", absIncludePath, err)
		}

		// Add to map
		configFileModTimes[absIncludePath] = info.ModTime()

		// Load the included file to check for nested includes
		var includedConfig Config
		_, err = toml.DecodeFile(absIncludePath, &includedConfig)
		if err != nil {
			return fmt.Errorf("failed to decode config file %s: %w", absIncludePath, err)
		}

		// Process nested includes
		if len(includedConfig.LDAPEnforcer.Includes) > 0 {
			err = addIncludedConfigFiles(&includedConfig, absIncludePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// CheckConfigFilesChanged checks if any of the config files have changed
// Returns true if any file has changed, false otherwise
func CheckConfigFilesChanged() (bool, error) {
	for filepath, oldModTime := range configFileModTimes {
		info, err := os.Stat(filepath)
		if err != nil {
			return false, fmt.Errorf("failed to get file info for %s: %w", filepath, err)
		}

		// Check if the file's modification time has changed
		if !info.ModTime().Equal(oldModTime) {
			return true, nil
		}
	}

	return false, nil
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
		c.LDAPEnforcer.MainLogLevel = val
	}
	if val := os.Getenv("LDAPENFORCER_LDAP_LOG_LEVEL"); val != "" {
		c.LDAPEnforcer.LDAPLogLevel = val
	}

	// Directory structure
	if val := os.Getenv("LDAPENFORCER_ENFORCED_PEOPLE_OU"); val != "" {
		c.LDAPEnforcer.EnforcedPeopleOU = val
	}
	if val := os.Getenv("LDAPENFORCER_ENFORCED_SVCACCT_OU"); val != "" {
		c.LDAPEnforcer.EnforcedSvcAcctOU = val
	}
	if val := os.Getenv("LDAPENFORCER_ENFORCED_GROUP_OU"); val != "" {
		c.LDAPEnforcer.EnforcedGroupOU = val
	}

	// Polling configuration
	if val := os.Getenv("LDAPENFORCER_POLL_CONFIG_INTERVAL"); val != "" {
		c.LDAPEnforcer.PollConfigInterval = val
	}
	if val := os.Getenv("LDAPENFORCER_POLL_LDAP_INTERVAL"); val != "" {
		c.LDAPEnforcer.PollLDAPInterval = val
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
	flags.String("log-level", "INFO", "Main log level (ERROR, WARN, INFO, DEBUG, TRACE)")
	flags.String("ldap-log-level", "INFO", "LDAP-specific log level (ERROR, WARN, INFO, DEBUG, TRACE)")
	flags.String("enforced-people-ou", "", "Full OU for enforced people")
	flags.String("enforced-svcacct-ou", "", "Full OU for enforced service accounts")
	flags.String("enforced-group-ou", "", "Full OU for enforced groups")
	flags.String("poll-config-interval", "10s", "Interval for --poll mode to check if the config file has changed and sync if so (recommended: \"10s\")")
	flags.String("poll-ldap-interval", "24h", "Interval for --poll mode to compare the config file to the LDAP server and sync if different (recommended: \"24h\")")
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
		c.LDAPEnforcer.MainLogLevel = logLevel
	}
	if ldapLogLevel, _ := flags.GetString("ldap-log-level"); ldapLogLevel != "" {
		c.LDAPEnforcer.LDAPLogLevel = ldapLogLevel
	}
	if enforcedPeopleOU, _ := flags.GetString("enforced-people-ou"); enforcedPeopleOU != "" {
		c.LDAPEnforcer.EnforcedPeopleOU = enforcedPeopleOU
	}
	if enforcedSvcAcctOU, _ := flags.GetString("enforced-svcacct-ou"); enforcedSvcAcctOU != "" {
		c.LDAPEnforcer.EnforcedSvcAcctOU = enforcedSvcAcctOU
	}
	if enforcedGroupOU, _ := flags.GetString("enforced-group-ou"); enforcedGroupOU != "" {
		c.LDAPEnforcer.EnforcedGroupOU = enforcedGroupOU
	}
	if pollConfigInterval, _ := flags.GetString("poll-config-interval"); pollConfigInterval != "" {
		c.LDAPEnforcer.PollConfigInterval = pollConfigInterval
	}
	if pollLDAPInterval, _ := flags.GetString("poll-ldap-interval"); pollLDAPInterval != "" {
		c.LDAPEnforcer.PollLDAPInterval = pollLDAPInterval
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

	if c.LDAPEnforcer.EnforcedPeopleOU == "" {
		return fmt.Errorf("enforced people OU is required")
	}
	if c.LDAPEnforcer.EnforcedSvcAcctOU == "" {
		return fmt.Errorf("enforced service account OU is required")
	}
	if c.LDAPEnforcer.EnforcedGroupOU == "" {
		return fmt.Errorf("enforced group OU is required")
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
