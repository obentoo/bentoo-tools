package main

import (
	"fmt"
	"os"

	"github.com/lucascouts/bentoo-tools/internal/common/config"
	"github.com/lucascouts/bentoo-tools/internal/overlay"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push committed changes to remote",
	Long:  `Push committed changes to the remote repository.`,
	Run:   runPush,
}

func init() {
	overlayCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	result, err := overlay.Push(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Message)
}
