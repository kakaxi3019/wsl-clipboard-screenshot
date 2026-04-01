#!/bin/bash
# Stop wsl-clipboard-screenshot daemon

set -e

BINARY="wsl-clipboard-screenshot"
PID_FILE="/tmp/.wsl-clipboard-screenshot.pid"

if command -v "$BINARY" &>/dev/null; then
    "$BINARY" stop
else
    # Fallback: kill by PID file
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            rm -f "$PID_FILE"
            echo "Stopped wsl-clipboard-screenshot (PID: $PID)"
        else
            rm -f "$PID_FILE"
            echo "Stale PID file removed"
        fi
    else
        # Try to find and kill any running instance
        PIDS=$(pgrep -f "wsl-clipboard-screenshot start" 2>/dev/null || true)
        if [ -n "$PIDS" ]; then
            echo "$PIDS" | xargs kill 2>/dev/null || true
            echo "Stopped wsl-clipboard-screenshot processes"
        else
            echo "wsl-clipboard-screenshot is not running"
        fi
    fi
fi
