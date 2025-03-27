// Generate documentation for the CLI using cobra/doc

package main

import (
	"log"
	"os"
	"time"

	"github.com/spf13/cobra/doc"

	"github.com/mrled/ldapenforcer/internal/cli/ldapenforcer"
)

func main() {
	if err := os.MkdirAll("./docs/cli", 0755); err != nil {
		log.Fatal(err)
	}
	if err := doc.GenMarkdownTree(ldapenforcer.RootCmd, "./docs/cli"); err != nil {
		log.Fatal(err)
	}

	// Man pages
	now := time.Now()
	header := &doc.GenManHeader{
		Title:   "ldapenforcer",
		Section: "1",
		Source:  "LDAPEnforcer Manual",
		Manual:  "LDAPEnforcer Manual",
		Date:    &now,
	}
	if err := os.MkdirAll("./docs/man", 0755); err != nil {
		log.Fatal(err)
	}
	if err := doc.GenManTree(ldapenforcer.RootCmd, header, "./docs/man"); err != nil {
		log.Fatal(err)
	}
}
