package poller

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kakaxi3019/wsl-clipboard-screenshot/internal/clipboard"
)

const (
	maxConsecutiveErrors = 5
	defaultInterval       = 250 * time.Millisecond
)

// Poller manages the clipboard polling loop
type Poller struct {
	client    clipboard.Clipboard
	interval  time.Duration
	outputDir string
	notify    bool
	logger    *log.Logger
}

// New creates a new Poller
func New(client clipboard.Clipboard, interval time.Duration, outputDir string, notify bool, logger *log.Logger) *Poller {
	if interval == 0 {
		interval = defaultInterval
	}
	return &Poller{
		client:    client,
		interval:  interval,
		outputDir: outputDir,
		notify:    notify,
		logger:    logger,
	}
}

// Run starts the polling loop until context is cancelled
func (p *Poller) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := p.poll(); err != nil {
				consecutiveErrors++
				p.logger.Printf("Poll error: %v (consecutive: %d)", err, consecutiveErrors)
				if consecutiveErrors >= maxConsecutiveErrors {
					p.logger.Printf("Circuit breaker: restarting PowerShell client")
					if err := p.restartClient(); err != nil {
						p.logger.Printf("Failed to restart client: %v", err)
						return err
					}
					consecutiveErrors = 0
				}
			} else {
				consecutiveErrors = 0
			}
		}
	}
}

func (p *Poller) poll() error {
	pngData, err := p.client.Check()
	if err != nil {
		return fmt.Errorf("Check failed: %w", err)
	}

	// No new image
	if pngData == nil {
		return nil
	}

	// Compute SHA256 content hash
	hash := HashBytes(pngData)
	filename := hash + ".png"
	filePath := filepath.Join(p.outputDir, filename)

	// Save if new (content-addressable dedup)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := os.WriteFile(filePath, pngData, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		p.logger.Printf("Screenshot saved: %s (%d bytes)", filename, len(pngData))
	} else {
		p.logger.Printf("Screenshot already exists: %s", filename)
	}

	// Update clipboard with 3 formats
	winPath, err := wslToWinPath(filePath)
	if err != nil {
		p.logger.Printf("Warning: wslpath failed, clipboard not updated: %v", err)
		return nil
	}

	if err := p.client.UpdateClipboard(filePath, winPath); err != nil {
		p.logger.Printf("Warning: clipboard update failed: %v", err)
		return nil
	}

	p.logger.Printf("Clipboard updated (WSL: %s)", filePath)

	// Send Windows notification if enabled
	if p.notify {
		notifyMsg := fmt.Sprintf("Path: %s", filePath)
		if err := p.client.Notify(notifyMsg); err != nil {
			p.logger.Printf("Warning: notification failed: %v", err)
		}
	}

	return nil
}

func (p *Poller) restartClient() error {
	p.client.Close()

	newClient, err := clipboard.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create new client: %w", err)
	}

	// Note: This replaces the interface value, not the pointer
	// The poller holds a clipboard.Client interface, so we need to update it
	p.client = newClient
	return nil
}

// wslToWinPath converts a WSL path to Windows path
// wslToWinPath converts a WSL path to a Windows path using wslpath -w
func wslToWinPath(wslPath string) (string, error) {
	out, err := exec.Command("wslpath", "-w", wslPath).Output()
	if err != nil {
		return "", fmt.Errorf("wslpath failed: %w", err)
	}
	// wslpath outputs with trailing newline
	return strings.TrimSpace(string(out)), nil
}
