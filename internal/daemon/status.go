package daemon

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Status struct {
	PID           int
	Uptime        time.Duration
	CPUPercent    float64
	MemoryRSSKB   int64
	ScreenshotCnt int
	OutputDir     string
}

func GetStatus(pidFile, outputDir string) (*Status, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read PID file: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return nil, fmt.Errorf("invalid PID file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process: %w", err)
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFile)
		return nil, fmt.Errorf("process not running (stale PID file)")
	}

	// Parse /proc/<pid>/stat for uptime and CPU
	statData, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return nil, fmt.Errorf("failed to read proc stat: %w", err)
	}

	stat := parseStat(string(statData))
	uptime := time.Now().Unix() - stat.starttime/100
	cpuTime := stat.utime + stat.stime

	// Parse /proc/uptime
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return nil, fmt.Errorf("failed to read uptime: %w", err)
	}
	var systemUptime float64
	fmt.Sscanf(string(uptimeData), "%f", &systemUptime)

	cpuPercent := float64(cpuTime) / systemUptime * 100 / float64(runtime.NumCPU())

	// Parse /proc/<pid>/status for memory
	statusData, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return nil, fmt.Errorf("failed to read proc status: %w", err)
	}

	var memKB int64
	for _, line := range strings.Split(string(statusData), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fmt.Sscanf(line, "VmRSS: %d kB", &memKB)
			break
		}
	}

	// Count screenshots
	matches, _ := filepath.Glob(filepath.Join(outputDir, "*.png"))
	screenshotCnt := len(matches)

	return &Status{
		PID:           pid,
		Uptime:        time.Duration(uptime) * time.Second,
		CPUPercent:    cpuPercent,
		MemoryRSSKB:   memKB,
		ScreenshotCnt: screenshotCnt,
		OutputDir:     outputDir,
	}, nil
}

type procStat struct {
	utime    int64
	stime    int64
	starttime int64
}

func parseStat(data string) procStat {
	// stat format: pid (comm) state ppid pgrp session tty_nr tpgid flags minflt cminflt majflt cmajflt utime stime ...
	// We need fields 14 (utime), 15 (stime), and 22 (starttime)
	// Comm can contain spaces and parentheses, so we need to find the last )
	fields := strings.Fields(data)
	if len(fields) < 22 {
		return procStat{}
	}

	utime, _ := strconv.ParseInt(fields[13], 10, 64)
	stime, _ := strconv.ParseInt(fields[14], 10, 64)
	starttime, _ := strconv.ParseInt(fields[21], 10, 64)

	return procStat{
		utime:    utime,
		stime:    stime,
		starttime: starttime,
	}
}

func (s *Status) Print(w io.Writer) {
	fmt.Fprintf(w, "PID:           %d\n", s.PID)
	fmt.Fprintf(w, "Uptime:        %s\n", s.Uptime)
	fmt.Fprintf(w, "CPU:           %.1f%%\n", s.CPUPercent)
	fmt.Fprintf(w, "Memory:        %d kB\n", s.MemoryRSSKB)
	fmt.Fprintf(w, "Screenshots:   %d\n", s.ScreenshotCnt)
	fmt.Fprintf(w, "Output Dir:    %s\n", s.OutputDir)
}
