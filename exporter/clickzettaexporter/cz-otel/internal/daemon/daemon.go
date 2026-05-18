// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DaemonInfo holds information about a running daemon process.
type DaemonInfo struct {
	PID       int
	StartTime time.Time
}

// Start launches the collector as a background daemon process.
// It starts the process, writes the PID file, and returns daemon info.
func Start(binaryPath, configPath, logPath, pidPath string) (*DaemonInfo, error) {
	// Ensure the log directory exists.
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	pid, err := startProcess(binaryPath, configPath, logPath)
	if err != nil {
		return nil, err
	}

	if err := WritePID(pidPath, pid); err != nil {
		// Try to stop the process we just started since we can't track it.
		_ = stopProcess(pid)
		return nil, fmt.Errorf("writing PID file after start: %w", err)
	}

	return &DaemonInfo{
		PID:       pid,
		StartTime: time.Now(),
	}, nil
}

// Stop stops the running daemon identified by the PID file.
// It reads the PID, stops the process, and removes the PID file.
func Stop(pidPath string) error {
	pid, err := ReadPID(pidPath)
	if err != nil {
		return fmt.Errorf("reading PID for stop: %w", err)
	}

	if err := stopProcess(pid); err != nil {
		return err
	}

	return RemovePID(pidPath)
}

// Status reads the PID file and checks if the daemon is running.
// Returns the daemon info, whether it's running, and any error.
func Status(pidPath string) (*DaemonInfo, bool, error) {
	pid, err := ReadPID(pidPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	running := isProcessRunning(pid)
	if !running {
		// Stale PID file — process is no longer running.
		return &DaemonInfo{PID: pid}, false, nil
	}

	return &DaemonInfo{
		PID:       pid,
		StartTime: time.Time{}, // Start time not tracked across restarts.
	}, true, nil
}

// IsRunning performs a quick check to determine if the daemon is running.
func IsRunning(pidPath string) bool {
	pid, err := ReadPID(pidPath)
	if err != nil {
		return false
	}
	return isProcessRunning(pid)
}
