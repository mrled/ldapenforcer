package ldapenforcer

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/mrled/ldapenforcer/internal/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of LDAPEnforcer",
	Long:  `All software has versions. This is LDAPEnforcer's.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("LDAPEnforcer version %s\n", version.GetVersion())
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}