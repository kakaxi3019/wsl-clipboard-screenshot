package platform

import (
	"fmt"
	"os"
	"strings"
)

const (
	procVersionPath = "/proc/version"
)

var (
	cached       bool
	isWSLCached   bool
	isWSL2Cached  bool
	procVersion   string
)

func init() {
	data, err := os.ReadFile(procVersionPath)
	if err != nil {
		cached = true // error state, keep cached
		return
	}
	procVersion = strings.ToLower(string(data))
	isWSLCached = strings.Contains(procVersion, "wsl") || strings.Contains(procVersion, "microsoft")
	isWSL2Cached = strings.Contains(procVersion, "wsl2") || strings.Contains(procVersion, "microsoft-standard-wsl2")
	cached = true
}

// IsWSL returns true if running in WSL
func IsWSL() bool {
	return isWSLCached
}

// IsWSL2 returns true if running in WSL2
func IsWSL2() bool {
	return isWSL2Cached
}

// RequireWSL exits if not running in WSL
func RequireWSL() {
	if !IsWSL() {
		fmt.Fprintln(os.Stderr, "Error: This tool only works in WSL (Windows Subsystem for Linux)")
		os.Exit(1)
	}
}
