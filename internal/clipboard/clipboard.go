package clipboard

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

//go:embed clipboard.ps1
var psScript string

// Clipboard defines the interface for clipboard operations
type Clipboard interface {
	Check() ([]byte, error)
	UpdateClipboard(wslPath, winPath string) error
	Notify(message string) error
	Close() error
}

// Client manages communication with the PowerShell STA process
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

var psPath = findPowerShell()

func findPowerShell() string {
	if _, err := os.Stat("/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"); err == nil {
		return "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
	}
	return "powershell.exe"
}

// NewClient creates a new PowerShell STA clipboard client
func NewClient() (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.Command(psPath,
		"-STA",
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-Command", psScript,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start PowerShell: %w", err)
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		ctx:    ctx,
		cancel: cancel,
	}

	// Wait for READY signal
	if err := c.waitForReady(); err != nil {
		c.Close()
		return nil, fmt.Errorf("PowerShell did not send READY: %w", err)
	}

	return c, nil
}

func (c *Client) waitForReady() error {
	resp, err := c.stdout.ReadString('\n')
	if err != nil {
		return err
	}
	resp = strings.TrimSpace(resp)
	if resp != RspReady {
		return fmt.Errorf("unexpected response: %s", resp)
	}
	return nil
}

// Close terminates the PowerShell process
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cancel()

	if c.stdin != nil {
		fmt.Fprintf(c.stdin, "%s\n", CmdExit)
		c.stdin.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Wait()
	}

	return nil
}

// Check queries the clipboard for a new image
// Returns the PNG bytes if a new image is available, nil otherwise
func (c *Client) Check() ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ctx.Err() != nil {
		return nil, c.ctx.Err()
	}

	// Send CHECK command
	if _, err := fmt.Fprintf(c.stdin, "%s\n", CmdCheck); err != nil {
		return nil, fmt.Errorf("failed to send CHECK: %w", err)
	}

	// Read response
	resp, err := c.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	resp = strings.TrimSpace(resp)

	switch resp {
	case RspNone:
		return nil, nil
	case RspImage:
		// Read base64 encoded image
		b64Data, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read image data: %w", err)
		}
		b64Data = strings.TrimSpace(b64Data)

		// Read END marker
		end, err := c.stdout.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read END marker: %w", err)
		}
		if strings.TrimSpace(end) != RspEnd {
			return nil, fmt.Errorf("missing END marker, got: %s", end)
		}

		return base64.StdEncoding.DecodeString(b64Data)
	case RspErr:
		errMsg, _ := c.stdout.ReadString('\n')
		return nil, fmt.Errorf("PowerShell error: %s", strings.TrimSpace(errMsg))
	default:
		return nil, fmt.Errorf("unexpected response: %s", resp)
	}
}

// UpdateClipboard sets three clipboard formats: CF_BITMAP, CF_UNICODETEXT, CF_HDROP
func (c *Client) UpdateClipboard(wslPath, winPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	cmd := fmt.Sprintf("%s|%s|%s", CmdUpdate, wslPath, winPath)
	if _, err := fmt.Fprintf(c.stdin, "%s\n", cmd); err != nil {
		return fmt.Errorf("failed to send UPDATE: %w", err)
	}

	resp, err := c.stdout.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read UPDATE response: %w", err)
	}
	resp = strings.TrimSpace(resp)

	if resp == RspOK {
		return nil
	}

	if strings.HasPrefix(resp, RspErr) {
		return fmt.Errorf("PowerShell UPDATE error: %s", resp[4:])
	}

	return fmt.Errorf("unexpected UPDATE response: %s", resp)
}

// Notify sends a Windows toast notification
func (c *Client) Notify(message string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	cmd := fmt.Sprintf("%s|%s", CmdNotify, message)
	if _, err := fmt.Fprintf(c.stdin, "%s\n", cmd); err != nil {
		return fmt.Errorf("failed to send NOTIFY: %w", err)
	}

	resp, err := c.stdout.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read NOTIFY response: %w", err)
	}
	resp = strings.TrimSpace(resp)

	if resp == RspOK {
		return nil
	}

	if strings.HasPrefix(resp, RspErr) {
		return fmt.Errorf("PowerShell NOTIFY error: %s", resp[4:])
	}

	return nil // OK is acceptable even if notification fails
}

// WaitForReadyWithTimeout waits for READY signal with timeout
func (c *Client) WaitForReadyWithTimeout(timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- c.waitForReady()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for READY")
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}
