package main

import (
	"fmt"
	"os"
)

// verbose, quiet and noColor carry the root's persistent flags to the run
// functions that read them (overlay_compare.go, overlay_autoupdate.go). They are
// written once per run, by the root's PersistentPreRun in root.go, and never
// bound directly to a flag — see newRootCmd for why that distinction matters.
var (
	verbose bool
	quiet   bool
	noColor bool
)

// osExit is a variable so tests can replace it to avoid process termination.
var osExit = os.Exit

// rootCmd is the process's own command tree: one call to the constructor that
// can build any number of them.
var rootCmd = newRootCmd()

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}
