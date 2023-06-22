package cmd

import (
	"context"
	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) int {
	rootCmd := &cobra.Command{
		Use:   "sloth",
		Short: "Sloth is an amazing scraper.",
	}

	rootCmd.AddCommand(WorkerCmd(ctx))

	if err := rootCmd.Execute(); err != nil {
		return 1
	}

	return 0
}
