package ldapenforcer

import (
	"fmt"

	"github.com/mrled/ldapenforcer/internal/ldap"
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

		// Create LDAP client
		client, err := ldap.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to create LDAP client: %w", err)
		}
		defer client.Close()

		// Run the sync
		fmt.Println("Starting LDAP synchronization...")

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			fmt.Println("Dry run mode - no changes will be made")
			return simulateSync(client)
		}

		err = client.SyncAll()
		if err != nil {
			return fmt.Errorf("synchronization failed: %w", err)
		}

		fmt.Println("LDAP synchronization completed successfully")
		return nil
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
