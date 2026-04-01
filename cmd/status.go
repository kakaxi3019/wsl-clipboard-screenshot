package cmd

import (
	"fmt"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/daemon"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status and diagnostics",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := flagOutput
		if outputDir == "" {
			outputDir = defaultOutput
		}

		status, err := daemon.GetStatus(daemon.PidFile, outputDir)
		if err != nil {
			// Friendly message when daemon is not running
			fmt.Fprintf(cmd.OutOrStdout(), "Daemon not running (PID file: %s)\n", daemon.PidFile)
			fmt.Fprintf(cmd.OutOrStdout(), "Start with: wsl-clipboard-screenshot start --daemon\n")
			return nil
		}

		status.Print(cmd.OutOrStdout())
		return nil
	},
}
