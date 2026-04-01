package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	flagInterval  time.Duration
	flagOutput    string
	flagDaemon    bool
	flagVerbose   bool
	flagNotify    bool
	defaultOutput = "/tmp/.wsl-clipboard-screenshot"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start monitoring the Windows clipboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := flagOutput
		if outputDir == "" {
			outputDir = defaultOutput
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		logger := log.New(os.Stdout, "", log.LstdFlags)
		if flagVerbose {
			logger.SetFlags(log.LstdFlags | log.Lshortfile)
		}

		ctx := cmd.Context()

		if flagDaemon {
			return daemon.Daemonize(ctx, flagInterval, outputDir, flagNotify, logger)
		}

		return daemon.Run(ctx, flagInterval, outputDir, flagNotify, logger)
	},
}

func init() {
	startCmd.Flags().DurationVar(&flagInterval, "interval", 250*time.Millisecond, "Polling interval")
	startCmd.Flags().StringVar(&flagOutput, "output", "", "Output directory (default: /tmp/.wsl-clipboard-screenshot)")
	startCmd.Flags().BoolVar(&flagDaemon, "daemon", false, "Run as background daemon")
	startCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")
	startCmd.Flags().BoolVar(&flagNotify, "notify", true, "Enable Windows notification on screenshot save")
}
