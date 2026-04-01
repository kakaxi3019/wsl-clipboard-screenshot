package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := daemon.RunningPID(daemon.PidFile)
		if err != nil {
			return fmt.Errorf("daemon not running: %w", err)
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find process: %w", err)
		}

		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}

		fmt.Printf("Sent SIGTERM to daemon (PID %d)\n", pid)
		return nil
	},
}
