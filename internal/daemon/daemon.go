package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/clipboard"
	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/poller"
)

const (
	PidFile  = "/tmp/.wsl-clipboard-screenshot.pid"
	LogFile  = "/tmp/.wsl-clipboard-screenshot.log"
)

// Cleanup removes screenshot files older than the specified number of days
func Cleanup(outputDir string, maxAgeDays int, logger *log.Logger) error {
	if maxAgeDays <= 0 {
		maxAgeDays = 7 // default 7 days
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist, nothing to clean
		}
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	var removedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".png" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(outputDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				logger.Printf("Warning: failed to remove old screenshot %s: %v", entry.Name(), err)
				continue
			}
			removedCount++
			logger.Printf("Removed old screenshot: %s", entry.Name())
		}
	}

	if removedCount > 0 {
		logger.Printf("Cleanup complete: removed %d old screenshots", removedCount)
	}

	return nil
}

func Daemonize(ctx context.Context, interval time.Duration, outputDir string, notify bool, logger *log.Logger) error {
	// Check if already running
	if pid, err := RunningPID(PidFile); err == nil {
		return fmt.Errorf("daemon already running with PID %d (pidfile: %s)", pid, PidFile)
	}

	// Create log file
	logF, err := os.OpenFile(LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logF.Close()

	// Spawn child process
	cmd := os.Args[0] + " start --interval=" + interval.String() + " --output=" + outputDir
	if !notify {
		cmd += " --notify=false"
	}
	if logger.Flags()&log.Lshortfile != 0 {
		cmd += " --verbose"
	}

	child, err := os.StartProcess("/bin/sh", []string{"/bin/sh", "-c", cmd}, &os.ProcAttr{
		Dir:   ".",
		Files: []*os.File{os.Stdin, logF, logF},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to spawn daemon: %w", err)
	}

	// Write PID file
	if err := os.WriteFile(PidFile, []byte(fmt.Sprintf("%d", child.Pid)), 0644); err != nil {
		child.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	logger.Printf("Daemon started with PID %d", child.Pid)
	return nil
}

func Run(ctx context.Context, interval time.Duration, outputDir string, notify bool, logger *log.Logger) error {
	// Cleanup old screenshots on startup
	if err := Cleanup(outputDir, 7, logger); err != nil {
		logger.Printf("Warning: startup cleanup failed: %v", err)
	}

	// Write PID file
	if err := os.WriteFile(PidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	defer os.Remove(PidFile)

	client, err := clipboard.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create clipboard client: %w", err)
	}
	defer client.Close()

	pllr := poller.New(client, interval, outputDir, notify, logger)
	return pllr.Run(ctx)
}

func RunningPID(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("invalid PID file content")
	}

	// Verify process exists
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, err
	}

	// Signal 0 checks if process exists without sending signal
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		// Process doesn't exist, stale PID file
		os.Remove(pidFile)
		return 0, fmt.Errorf("stale PID file")
	}

	return pid, nil
}
