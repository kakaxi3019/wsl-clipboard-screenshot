package cmd

import (
	"context"
	"fmt"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/platform"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "wsl-clipboard-screenshot",
	Short: "Monitor Windows clipboard for screenshots in WSL",
	Long: `A daemon that monitors the Windows clipboard for screenshots,
automatically saving them to a file and outputting the path when pasting in WSL.`,
	Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		platform.RequireWSL()
	},
}

func Execute(ctx context.Context) error {
	rootCmd.AddCommand(startCmd, stopCmd, statusCmd, updateCmd)
	return rootCmd.ExecuteContext(ctx)
}
