package ldapenforcer

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// configShowCmd represents the config-show command
var configShowCmd = &cobra.Command{
	Use:   "config-show",
	Short: "Display the current configuration",
	Long: `Display the current configuration in TOML format after
all sources (defaults, config file, environment variables,
and command line flags) have been applied.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("no configuration loaded")
		}

		// Create a new buffer to hold the TOML output
		var buf bytes.Buffer

		// Encode the config as TOML
		// The private field processedIncludes will be automatically excluded
		// since it's not exported and the TOML encoder only encodes exported fields
		encoder := toml.NewEncoder(&buf)
		err := encoder.Encode(cfg)
		if err != nil {
			return fmt.Errorf("error encoding configuration: %w", err)
		}

		// Print the configuration to stdout
		_, err = io.Copy(os.Stdout, &buf)
		if err != nil {
			return fmt.Errorf("error writing configuration: %w", err)
		}

		return nil
	},
}

func init() {
	// Add the config-show command to the root command
	RootCmd.AddCommand(configShowCmd)
}
