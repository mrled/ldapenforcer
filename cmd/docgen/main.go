// Generate documentation for the CLI using cobra/doc

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/mrled/ldapenforcer/internal/cli/ldapenforcer"
)

func filePrepender(filename string) string {
	// Extract the base name (command name) without the .md extension.
	fileBaseName := strings.TrimSuffix(filepath.Base(filename), ".md")

	// Cobra generates files with underscores when the command has subcommands,
	// e.g. `git init` -> `git_init.md`.
	// For our titles, we want spaces instead of underscores.
	title := strings.Replace(fileBaseName, "_", " ", -1)

	return fmt.Sprintf("+++\ntitle = \"%s\"\n+++\n\n", title)
}

func linkHandler(name string) string {
	return name
}

func generateMarkdown(path string) (err error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}

	// The simplest way to generate markdown files:
	// if err := doc.GenMarkdownTree(ldapenforcer.RootCmd, path); err != nil {
	// 	return err
	// }
	//

	// Generate markdown files with custom file prepender that contains TOML frontmatter:
	if err := doc.GenMarkdownTreeCustom(ldapenforcer.RootCmd, path, filePrepender, linkHandler); err != nil {
		return err
	}

	return nil
}

func generateManual(path string) (err error) {
	now := time.Now()
	manHeader := &doc.GenManHeader{
		Title:   "ldapenforcer",
		Section: "1",
		Source:  "LDAPEnforcer Manual",
		Manual:  "LDAPEnforcer Manual",
		Date:    &now,
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	if err := doc.GenManTree(ldapenforcer.RootCmd, manHeader, path); err != nil {
		return err
	}
	return nil
}

func main() {
	var markdownPath string
	var manPath string

	var rootCmd = &cobra.Command{
		Use:   "docgen",
		Short: "Generate documentation for LDAPEnforcer",
		RunE: func(cmd *cobra.Command, args []string) error {
			if markdownPath != "" {
				if err := generateMarkdown(markdownPath); err != nil {
					return err
				}
			}
			if manPath != "" {
				if err := generateManual(manPath); err != nil {
					return err
				}
			}
			if markdownPath == "" && manPath == "" {
				return cmd.Help()
			}
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&markdownPath, "markdown-path", "m", "", "Path to save markdown files")
	rootCmd.Flags().StringVarP(&manPath, "man-path", "t", "", "Path to save troff-formatted manual pages")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
