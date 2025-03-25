package ldapenforcer

import (
	"fmt"

	"github.com/mrled/ldapenforcer/internal/version"
	"github.com/spf13/cobra"
)

// flagRaw determines whether to output just the version string (for scripting usage)
var flagRaw bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of LDAPEnforcer",
	Long:  `All software has versions. This is LDAPEnforcer's.`,
	Run: func(cmd *cobra.Command, args []string) {
		if flagRaw {
			fmt.Println(version.GetVersion())
		} else {
			fmt.Printf("LDAPEnforcer version %s\n", version.GetVersion())
		}
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&flagRaw, "raw", "r", false, "Print only the raw version number")
	RootCmd.AddCommand(versionCmd)
}
