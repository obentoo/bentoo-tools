package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bentoo",
	Short: "Bentoo Linux tools",
	Long:  `A collection of tools for managing Bentoo Linux overlay and packages.`,
}

var overlayCmd = &cobra.Command{
	Use:   "overlay",
	Short: "Manage the Bentoo overlay repository",
	Long:  `Commands for managing the Bentoo overlay repository including adding files, checking status, committing changes, and pushing to remote.`,
}

func init() {
	rootCmd.AddCommand(overlayCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
