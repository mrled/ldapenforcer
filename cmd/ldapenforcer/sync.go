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

		// Check if we should run in polling mode
		pollInterval := cfg.LDAPEnforcer.ConfigPollInterval
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// If polling is enabled, run continuously
		if pollInterval > 0 {
			// Limit minimum poll interval
			if pollInterval < 1 {
				fmt.Println("Warning: Minimum poll interval is 1 second, using 1 second")
				pollInterval = 1
			}

			// Initialize file monitoring
			err := config.InitConfigFileMonitoring(cfg)
			if err != nil {
				return fmt.Errorf("failed to initialize config file monitoring: %w", err)
			}

			fmt.Printf("Starting continuous sync with config poll interval of %d seconds\n", pollInterval)
			fmt.Println("Press Ctrl+C to stop")

			ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
			defer ticker.Stop()

			// Setup signal handling for graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			// Run first sync immediately - retry like we do in polling mode
			for {
				err := runSync(cfg, dryRun)
				if err != nil {
					fmt.Printf("Error during initial sync: %v\n", err)
					fmt.Printf("Retrying in %d seconds...\n", pollInterval)

					// Wait for either the retry interval or a signal
					select {
					case <-time.After(time.Duration(pollInterval) * time.Second):
						// Continue with retry
						continue
					case sig := <-sigChan:
						// User pressed Ctrl+C
						fmt.Printf("\nReceived signal %v during retry wait, shutting down...\n", sig)
						return nil
					}
				} else {
					// Successfully connected and synced
					break
				}
			}

			// Main polling loop
			for {
				select {
				case <-ticker.C:
					// Check if config files have changed
					changed, err := config.CheckConfigFilesChanged()
					if err != nil {
						fmt.Printf("Error checking config files: %v\n", err)
						continue
					}

					if changed {
						fmt.Println("Config files changed, reloading configuration...")

						// Reload configuration
						newCfg, err := config.LoadConfig(config.GetMainConfigFile())
						if err != nil {
							fmt.Printf("Error reloading config: %v\n", err)
							continue
						}

						// Merge with command line flags
						newCfg.MergeWithFlags(cmd.Flags())

						// Set log levels from the new config
						if newCfg.LDAPEnforcer.Logging.Level != "" {
							level, err := logging.ParseLevel(newCfg.LDAPEnforcer.Logging.Level)
							if err == nil {
								logging.DefaultLogger.SetLevel(level)
								logging.DefaultLogger.Debug("Main log level set to %s", logging.GetLevelName(level))
							}
						}

						// Set LDAP log level
						if newCfg.LDAPEnforcer.Logging.LDAP.Level != "" {
							level, err := logging.ParseLevel(newCfg.LDAPEnforcer.Logging.LDAP.Level)
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
							fmt.Printf("Error reinitializing file monitoring: %v\n", err)
							continue
						}

						// Run sync with new configuration
						if err := runSync(cfg, dryRun); err != nil {
							fmt.Printf("Error during sync after config reload: %v\n", err)
						}
					} else {
						// Optionally run sync even without config changes
						if err := runSync(cfg, dryRun); err != nil {
							fmt.Printf("Error during periodic sync: %v\n", err)
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
		defer client.Close()

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
		defer client.Close()

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
		defer client.Close()

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

// runSync runs a single synchronization operation
func runSync(cfg *config.Config, dryRun bool) error {
	logging.DefaultLogger.Debug("Starting LDAP synchronization...")

	// Create LDAP client
	client, err := ldap.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}
	defer client.Close()

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
	syncPersonCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	syncSvcAcctCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
	syncGroupCmd.Flags().Bool("dry-run", false, "Perform a dry run without making changes")
}
