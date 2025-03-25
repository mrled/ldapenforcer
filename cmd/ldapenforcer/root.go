package ldapenforcer

import (
	"fmt"
	"os"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/mrled/ldapenforcer/internal/model"
	"github.com/spf13/cobra"
)

var (
	// Used for flags
	cfgFile string
	cfg     *config.Config
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "ldapenforcer",
	Short: "LDAPEnforcer is a tool for enforcing LDAP policies",
	Long: `LDAPEnforcer is a command line tool for managing and
enforcing policies on LDAP directories.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip loading config for the version command
		if cmd.Name() == "version" && cmd.Parent().Name() == "ldapenforcer" {
			return nil
		}

		var err error

		// If config file specified, load it (this includes the defaults and the config file values)
		if cfgFile != "" {
			cfg, err = config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("error loading config file: %w", err)
			}
		} else {
			// If no config file, create an empty config with defaults
			cfg = createEmptyConfig()
			cfg.LDAPEnforcer.MainLogLevel = "INFO"
			cfg.LDAPEnforcer.LDAPLogLevel = "INFO"
		}

		// Apply configurations in order of increasing precedence:
		// Config file was loaded first (or defaults if no config)
		// Now apply: Environment variables (higher precedence than config)
		cfg.MergeWithEnv()

		// Finally, apply command line flags (highest precedence)
		cfg.MergeWithFlags(cmd.Flags())

		// Initialize main logging system
		mainLogLevel := logging.InfoLevel // Default to INFO level

		// Configure logging from the configuration
		if cfg.LDAPEnforcer.MainLogLevel != "" {
			level, err := logging.ParseLevel(cfg.LDAPEnforcer.MainLogLevel)
			if err == nil {
				mainLogLevel = level
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Invalid main log level '%s', using INFO level instead\n", cfg.LDAPEnforcer.MainLogLevel)
			}
		}

		// Set main logging level
		logging.DefaultLogger.SetLevel(mainLogLevel)
		logging.DefaultLogger.Debug("Main log level set to %s", logging.GetLevelName(mainLogLevel))

		// Initialize LDAP-specific logging
		ldapLogLevel := mainLogLevel

		// If LDAP-specific level is configured, use it instead
		if cfg.LDAPEnforcer.LDAPLogLevel != "" {
			level, err := logging.ParseLevel(cfg.LDAPEnforcer.LDAPLogLevel)
			if err == nil {
				ldapLogLevel = level
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Invalid LDAP log level '%s', using main log level instead\n", cfg.LDAPEnforcer.LDAPLogLevel)
			}
		}

		// Set LDAP logging level
		logging.LDAPProtocolLogger.SetLevel(ldapLogLevel)

		if ldapLogLevel != mainLogLevel {
			logging.DefaultLogger.Debug("LDAP log level set to %s", logging.GetLevelName(ldapLogLevel))
		}

		// Print formatted configuration
		logging.DefaultLogger.Debug("Configuration loaded: URI=%s, BindDN=%s, EnforcedPeopleOU=%s, EnforcedSvcAcctOU=%s, EnforcedGroupOU=%s, ConfigPollInterval=%d",
			cfg.LDAPEnforcer.URI,
			cfg.LDAPEnforcer.BindDN,
			cfg.LDAPEnforcer.EnforcedPeopleOU,
			cfg.LDAPEnforcer.EnforcedSvcAcctOU,
			cfg.LDAPEnforcer.EnforcedGroupOU,
			cfg.LDAPEnforcer.ConfigPollInterval)
		return nil
	},
}

// createEmptyConfig creates a new empty config with initialized maps
func createEmptyConfig() *config.Config {
	// Create a new config with initialized maps
	emptyConfig := &config.Config{
		LDAPEnforcer: config.LDAPEnforcerConfig{
			Person:   make(map[string]*model.Person),
			SvcAcct:  make(map[string]*model.SvcAcct),
			Group:    make(map[string]*model.Group),
			Includes: make([]string, 0),
		},
	}

	return emptyConfig
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Define flags for the root command
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path")

	// Add all config flags
	config.AddFlags(RootCmd.PersistentFlags())
}
