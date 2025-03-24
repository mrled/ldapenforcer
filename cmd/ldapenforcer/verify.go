package ldapenforcer

import (
	"fmt"

	ldapv3 "github.com/go-ldap/ldap/v3"
	"github.com/mrled/ldapenforcer/internal/ldap"
	"github.com/spf13/cobra"
)

// verifyCmd represents the verify command
var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify LDAP against configuration",
	Long:  `Verifies that LDAP directory matches the current configuration without making changes.`,
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

		// Check if LDAP is accessible
		fmt.Println("Verifying LDAP connection...")
		_, err = client.Search("", "(objectClass=*)", []string{"namingContexts"})
		if err != nil {
			return fmt.Errorf("LDAP connection failed: %w", err)
		}

		fmt.Println("LDAP connection successful")

		// Check people
		fmt.Println("\nVerifying people...")
		for uid := range cfg.LDAPEnforcer.Person {
			dn := client.PersonToDN(uid)
			checkEntity(client, dn, "person")
		}

		// Check service accounts
		fmt.Println("\nVerifying service accounts...")
		for uid := range cfg.LDAPEnforcer.SvcAcct {
			dn := client.SvcAcctToDN(uid)
			checkEntity(client, dn, "service account")
		}

		// Check groups
		fmt.Println("\nVerifying groups...")
		for groupname := range cfg.LDAPEnforcer.Group {
			dn := client.GroupToDN(groupname)
			checkEntity(client, dn, "group")
		}

		fmt.Println("\nVerification complete")
		return nil
	},
}

// checkOU checks if an OU exists in LDAP
func checkOU(client *ldap.Client, dn string) {
	exists, _ := client.EntryExists(dn)
	if exists {
		fmt.Printf("✓ %s exists\n", dn)
	} else {
		fmt.Printf("✗ %s does not exist\n", dn)
	}
}

// checkEntity checks if an entity exists in LDAP
func checkEntity(client *ldap.Client, dn string, entityType string) {
	exists, _ := client.EntryExists(dn)
	if exists {
		fmt.Printf("✓ %s %s exists\n", entityType, dn)
	} else {
		fmt.Printf("✗ %s %s does not exist\n", entityType, dn)
	}
}

// verifyPersonCmd represents the verify-person command
var verifyPersonCmd = &cobra.Command{
	Use:   "verify-person [uid]",
	Short: "Verify a specific person",
	Long:  `Verifies that a specific person in LDAP matches the configuration.`,
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

		// Check if person exists
		dn := client.PersonToDN(uid)
		exists, err := client.EntryExists(dn)
		if err != nil {
			return fmt.Errorf("failed to check if person exists: %w", err)
		}

		if !exists {
			fmt.Printf("Person %s does not exist in LDAP\n", uid)
			return nil
		}

		// Get person from LDAP
		entry, err := client.GetEntity(dn, []string{"*"})
		if err != nil {
			return fmt.Errorf("failed to get person from LDAP: %w", err)
		}

		// Compare attributes
		fmt.Printf("Verifying person: %s\n", uid)

		// Required attributes
		verifyAttribute(entry, "cn", []string{person.CN})
		verifyAttribute(entry, "sn", []string{person.GetSN()})

		// Optional attributes
		if person.GivenName != "" {
			verifyAttribute(entry, "givenName", []string{person.GivenName})
		}
		if person.Mail != "" {
			verifyAttribute(entry, "mail", []string{person.Mail})
		}

		// POSIX attributes
		if person.IsPosix() {
			verifyAttribute(entry, "uidNumber", []string{fmt.Sprintf("%d", person.GetUIDNumber())})
			verifyAttribute(entry, "gidNumber", []string{fmt.Sprintf("%d", person.GetGIDNumber())})
		}

		return nil
	},
}

// verifyAttribute checks if an attribute in an LDAP entry matches the expected values
func verifyAttribute(entry *ldapv3.Entry, attrName string, expectedValues []string) {
	attribute := entry.GetAttributeValues(attrName)

	// Check if attribute exists
	if len(attribute) == 0 {
		if len(expectedValues) == 0 {
			fmt.Printf("✓ %s: not set (as expected)\n", attrName)
		} else {
			fmt.Printf("✗ %s: not set (expected %v)\n", attrName, expectedValues)
		}
		return
	}

	// Check if values match
	match := len(attribute) == len(expectedValues)
	if match {
		for i, val := range attribute {
			if val != expectedValues[i] {
				match = false
				break
			}
		}
	}

	if match {
		fmt.Printf("✓ %s: %v\n", attrName, attribute)
	} else {
		fmt.Printf("✗ %s: %v (expected %v)\n", attrName, attribute, expectedValues)
	}
}

func init() {
	// Add commands to the root command
	RootCmd.AddCommand(verifyCmd)

	// Add subcommands to the verify command
	verifyCmd.AddCommand(verifyPersonCmd)
}
