package ldapenforcer

import (
	"fmt"
	"os"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/logging"
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

		// If config file specified, load it
		if cfgFile != "" {
			cfg, err = config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("error loading config file: %w", err)
			}

			// Merge command line flags with config
			cfg.MergeWithFlags(cmd.Flags())
		} else {
			// If no config file, create an empty config with just flags
			cfg = &config.Config{}
			cfg.MergeWithFlags(cmd.Flags())
		}

		// Initialize main logging system
		mainLogLevel := logging.ErrorLevel

		// Configure logging from the configuration
		if cfg.LDAPEnforcer.Logging.Level != "" {
			level, err := logging.ParseLevel(cfg.LDAPEnforcer.Logging.Level)
			if err == nil {
				mainLogLevel = level
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Invalid main log level '%s', using ERROR level instead\n", cfg.LDAPEnforcer.Logging.Level)
			}
		}

		// Set main logging level
		logging.DefaultLogger.SetLevel(mainLogLevel)
		logging.DefaultLogger.Debug("Main log level set to %s", logging.GetLevelName(mainLogLevel))

		// Initialize LDAP-specific logging
		ldapLogLevel := mainLogLevel

		// If LDAP-specific level is configured, use it instead
		if cfg.LDAPEnforcer.Logging.LDAP.Level != "" {
			level, err := logging.ParseLevel(cfg.LDAPEnforcer.Logging.LDAP.Level)
			if err == nil {
				ldapLogLevel = level
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Invalid LDAP log level '%s', using main log level instead\n", cfg.LDAPEnforcer.Logging.LDAP.Level)
			}
		}

		// Set LDAP logging level
		logging.LDAPProtocolLogger.SetLevel(ldapLogLevel)

		if ldapLogLevel != mainLogLevel {
			logging.DefaultLogger.Debug("LDAP log level set to %s", logging.GetLevelName(ldapLogLevel))
		}

		return nil
	},
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
