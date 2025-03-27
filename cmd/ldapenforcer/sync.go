package ldapenforcer

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrled/ldapenforcer/internal/config"
	"github.com/mrled/ldapenforcer/internal/ldap"
	"github.com/mrled/ldapenforcer/internal/logging"
	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize LDAP with configuration",
	Long:  `Synchronizes LDAP directory with the current configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		// Validate configuration
		err := cfg.Validate()
		if err != nil {
			return fmt.Errorf("configuration is invalid: %w", err)
		}

		// Get command flags
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		pollEnabled, _ := cmd.Flags().GetBool("poll")
		pollConfigIntervalStr, _ := cmd.Flags().GetString("poll-config-interval")
		pollLDAPIntervalStr, _ := cmd.Flags().GetString("poll-ldap-interval")

		// If polling is enabled, run continuously
		if pollEnabled {
			// Parse the polling intervals
			pollConfigInterval, err := time.ParseDuration(pollConfigIntervalStr)
			if err != nil {
				return fmt.Errorf("invalid poll-config-interval: %w", err)
			}
			if pollConfigInterval < time.Second {
				fmt.Println("Warning: Minimum config poll interval is 1 second, using 1 second")
				pollConfigInterval = time.Second
			}

			pollLDAPInterval, err := time.ParseDuration(pollLDAPIntervalStr)
			if err != nil {
				return fmt.Errorf("invalid poll-ldap-interval: %w", err)
			}
			if pollLDAPInterval < time.Minute {
				fmt.Println("Warning: Minimum LDAP poll interval is 1 minute, using 1 minute")
				pollLDAPInterval = time.Minute
			}

			// Initialize file monitoring
			err = config.InitConfigFileMonitoring(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize config file monitoring: %w", err)
			}

			fmt.Printf("Starting continuous sync:\n")
			fmt.Printf("  - Config file check interval: %s\n", pollConfigInterval)
			fmt.Printf("  - LDAP server check interval: %s\n", pollLDAPInterval)
			fmt.Println("Press Ctrl+C to stop")

			// Create two tickers for different intervals
			configTicker := time.NewTicker(pollConfigInterval)
			ldapTicker := time.NewTicker(pollLDAPInterval)
			defer configTicker.Stop()
			defer ldapTicker.Stop()

			// Track last LDAP sync time
			lastLDAPSync := time.Now()

			// Setup signal handling for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Run first sync immediately - retry like we do in polling mode
			for {
				err := runSync(cfg, dryRun)
				if err != nil {
					fmt.Printf("Error during initial sync: %v\n", err)
					fmt.Printf("Retrying in %s...\n", pollConfigInterval)

					// Wait for either the retry interval or a signal
					select {
					case <-time.After(pollConfigInterval):
						// Continue with retry
						continue
					case sig := <-sigChan:
						// User pressed Ctrl+C
						fmt.Printf("\nReceived signal %v during retry wait, shutting down...\n", sig)
						return nil
					}
				} else {
					// Successfully connected and synced
					lastLDAPSync = time.Now()
					break
				}
			}

			// Main polling loop
			for {
				select {
				case <-configTicker.C:
					// Check if config files have changed
					changed, err := config.CheckConfigFilesChanged()
					if err != nil {
						fmt.Printf("Error checking config files: %v\n", err)
						continue
					}
					if !changed {
						logging.DefaultLogger.Trace("Config files haven't changed, skipping LDAP operations")
						continue
					}
					fmt.Println("Config files changed, reloading configuration...")
					if reloadConfig(cmd, dryRun) == nil {
						// Run sync with new configuration
						if err := runSync(cfg, dryRun); err == nil {
							lastLDAPSync = time.Time{}
						} else {
							fmt.Printf("Error during sync triggered by config reload: %v\n", err)
						}
					} else {
						fmt.Printf("Error during config reload: %v\n", err)
					}

				case <-ldapTicker.C:
					timeSinceLastSync := time.Since(lastLDAPSync)
					if timeSinceLastSync >= pollLDAPInterval {
						logging.DefaultLogger.Info("Running periodic LDAP sync (it's been %s since last sync)", timeSinceLastSync)

						// Run sync operation to ensure LDAP is in sync with config
						if err := runSync(cfg, dryRun); err != nil {
							fmt.Printf("Error during sync triggered by LDAP poll interval: %v\n", err)
						} else {
							lastLDAPSync = time.Now()
						}
					}

				case <-sigChan:
					fmt.Println("\nReceived interrupt signal, shutting down...")
					return nil
				}
			}
		} else {
			// Run once without polling
			return runSync(cfg, dryRun)
		}
	},
}

// syncPersonCmd represents the sync-person command
var syncPersonCmd = &cobra.Command{
	Use:   "sync-person [uid]",
	Short: "Synchronize a specific person",
	Long:  `Synchronizes a specific person in LDAP with the configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		uid := args[0]
		person, ok := cfg.LDAPEnforcer.Person[uid]
		if !ok {
			return fmt.Errorf("person %s not found in configuration", uid)
		}

		// Create LDAP client
		client, err := ldap.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create LDAP client: %w", err)
		}

		// Ensure connection is closed at the end of the operation
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				logging.DefaultLogger.Warn("Error closing LDAP connection: %v", closeErr)
			}
		}()

		// Ensure OUs exist
		err = client.EnsureManagedOUsExist()
		if err != nil {
			return fmt.Errorf("failed to ensure OUs exist: %w", err)
		}

		// Sync the person
		fmt.Printf("Synchronizing person: %s\n", uid)

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Println("Dry run mode - no changes will be made")
			dn := client.PersonToDN(uid)
			attrs := ldap.GetPersonAttributes(person)
			fmt.Printf("Would create/update person: %s\n", dn)
			for attr, values := range attrs {
				fmt.Printf("  %s: %v\n", attr, values)
			}
			return nil
		}

		err = client.SyncPerson(uid, person)
		if err != nil {
			return fmt.Errorf("failed to sync person: %w", err)
		}

		fmt.Printf("Person %s synchronized successfully\n", uid)
		return nil
	},
}

// syncSvcAcctCmd represents the sync-svcacct command
var syncSvcAcctCmd = &cobra.Command{
	Use:   "sync-svcacct [uid]",
	Short: "Synchronize a specific service account",
	Long:  `Synchronizes a specific service account in LDAP with the configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		uid := args[0]
		svcacct, ok := cfg.LDAPEnforcer.SvcAcct[uid]
		if !ok {
			return fmt.Errorf("service account %s not found in configuration", uid)
		}

		// Create LDAP client
		client, err := ldap.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create LDAP client: %w", err)
		}

		// Ensure connection is closed at the end of the operation
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				logging.DefaultLogger.Warn("Error closing LDAP connection: %v", closeErr)
			}
		}()

		// Ensure OUs exist
		err = client.EnsureManagedOUsExist()
		if err != nil {
			return fmt.Errorf("failed to ensure OUs exist: %w", err)
		}

		// Sync the service account
		fmt.Printf("Synchronizing service account: %s\n", uid)

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Println("Dry run mode - no changes will be made")
			dn := client.SvcAcctToDN(uid)
			attrs := ldap.GetSvcAcctAttributes(svcacct)
			fmt.Printf("Would create/update service account: %s\n", dn)
			for attr, values := range attrs {
				fmt.Printf("  %s: %v\n", attr, values)
			}
			return nil
		}

		err = client.SyncSvcAcct(uid, svcacct)
		if err != nil {
			return fmt.Errorf("failed to sync service account: %w", err)
		}

		fmt.Printf("Service account %s synchronized successfully\n", uid)
		return nil
	},
}

// syncGroupCmd represents the sync-group command
var syncGroupCmd = &cobra.Command{
	Use:   "sync-group [groupname]",
	Short: "Synchronize a specific group",
	Long:  `Synchronizes a specific group in LDAP with the configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		groupname := args[0]
		group, ok := cfg.LDAPEnforcer.Group[groupname]
		if !ok {
			return fmt.Errorf("group %s not found in configuration", groupname)
		}

		// Create LDAP client
		client, err := ldap.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create LDAP client: %w", err)
		}

		// Ensure connection is closed at the end of the operation
		defer func() {
			if closeErr := client.Close(); closeErr != nil {
				logging.DefaultLogger.Warn("Error closing LDAP connection: %v", closeErr)
			}
		}()

		// Ensure OUs exist
		err = client.EnsureManagedOUsExist()
		if err != nil {
			return fmt.Errorf("failed to ensure OUs exist: %w", err)
		}

		// Sync the group
		fmt.Printf("Synchronizing group: %s\n", groupname)

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Println("Dry run mode - no changes will be made")
			dn := client.GroupToDN(groupname)
			attrs, err := client.GetGroupAttributes(groupname, group)
			if err != nil {
				return fmt.Errorf("failed to get group attributes: %w", err)
			}
			fmt.Printf("Would create/update group: %s\n", dn)
			for attr, values := range attrs {
				fmt.Printf("  %s: %v\n", attr, values)
			}
			return nil
		}

		err = client.SyncGroup(groupname, group)
		if err != nil {
			return fmt.Errorf("failed to sync group: %w", err)
		}

		fmt.Printf("Group %s synchronized successfully\n", groupname)
		return nil
	},
}

// simulateSync simulates a sync operation without making changes
func simulateSync(client *ldap.Client) error {
	// Show what would be done
	fmt.Println("Dry run summary:")
	fmt.Printf("- Would ensure managed OUs exist\n")
	fmt.Printf("- Would sync %d people\n", len(cfg.LDAPEnforcer.Person))
	fmt.Printf("- Would sync %d service accounts\n", len(cfg.LDAPEnforcer.SvcAcct))
	fmt.Printf("- Would sync %d groups\n", len(cfg.LDAPEnforcer.Group))

	return nil
}

// reloadConfig reloads configuration from disk
func reloadConfig(cmd *cobra.Command, dryRun bool) error {
	// Reload configuration
	newCfg, err := config.LoadConfig(config.GetMainConfigFile())
	if err != nil {
		return fmt.Errorf("error reloading config: %w", err)
	}

	// Merge with command line flags
	newCfg.MergeWithFlags(cmd.Flags())

	// Set log levels from the new config
	if newCfg.LDAPEnforcer.MainLogLevel != "" {
		level, err := logging.ParseLevel(newCfg.LDAPEnforcer.MainLogLevel)
		if err == nil {
			logging.DefaultLogger.SetLevel(level)
			logging.DefaultLogger.Debug("Main log level set to %s", logging.GetLevelName(level))
		}
	}

	// Set LDAP log level
	if newCfg.LDAPEnforcer.LDAPLogLevel != "" {
		level, err := logging.ParseLevel(newCfg.LDAPEnforcer.LDAPLogLevel)
		if err == nil {
			logging.LDAPProtocolLogger.SetLevel(level)
			logging.DefaultLogger.Debug("LDAP log level set to %s", logging.GetLevelName(level))
		}
	}

	// Update the global config
	cfg = newCfg

	// Reinitialize file monitoring with new config
	err = config.InitConfigFileMonitoring(cfg)
	if err != nil {
		return fmt.Errorf("error reinitializing file monitoring: %w", err)
	}

	// Run sync with new configuration
	if err := runSync(cfg, dryRun); err != nil {
		return fmt.Errorf("error during sync after config reload: %w", err)
	}

	return nil
}

// runSync runs a single synchronization operation
func runSync(cfg *config.Config, dryRun bool) error {
	logging.DefaultLogger.Debug("Starting LDAP synchronization...")

	// Create LDAP client
	client, err := ldap.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}

	// Ensure connection is closed at the end of the operation
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			logging.DefaultLogger.Warn("Error closing LDAP connection: %v", closeErr)
		}
	}()

	// Run the sync
	if dryRun {
		logging.DefaultLogger.Info("Dry run mode - no changes will be made")
		return simulateSync(client)
	}

	err = client.SyncAll()
	if err != nil {
		return fmt.Errorf("synchronization failed: %w", err)
	}

	logging.DefaultLogger.Info("LDAP synchronization completed successfully")
	return nil
}

func init() {
	// Add commands to the root command
	RootCmd.AddCommand(syncCmd)

	// Add subcommands to the sync command
	syncCmd.AddCommand(syncPersonCmd)
	syncCmd.AddCommand(syncSvcAcctCmd)
	syncCmd.AddCommand(syncGroupCmd)

	// Add flags
	syncCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	syncCmd.Flags().Bool("poll", false, "Enable polling mode to continuously check for config changes")
	syncCmd.Flags().String("poll-config-interval", "10s", "Interval for --poll mode to check if the config file has changed and sync if so (recommended: \"10s\")")
	syncCmd.Flags().String("poll-ldap-interval", "24h", "Interval for --poll mode to compare the config file to the LDAP server and sync if different (recommended: \"24h\")")
	syncPersonCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	syncSvcAcctCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	syncGroupCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
}
