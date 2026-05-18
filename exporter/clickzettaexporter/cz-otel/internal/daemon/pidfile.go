// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const pidFilePermissions = 0600

// WritePID writes the given PID to the specified file with 0600 permissions.
func WritePID(path string, pid int) error {
	data := []byte(strconv.Itoa(pid))
	if err := os.WriteFile(path, data, pidFilePermissions); err != nil {
		return fmt.Errorf("writing PID file: %w", err)
	}
	return nil
}

// ReadPID reads and returns the PID from the specified file.
// Returns an error if the file doesn't exist or contains invalid data.
func ReadPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading PID file: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file %s: %w", path, err)
	}

	if pid <= 0 {
		return 0, fmt.Errorf("invalid PID value: %d", pid)
	}

	return pid, nil
}

// RemovePID removes the PID file at the specified path.
// Returns nil if the file doesn't exist.
func RemovePID(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing PID file: %w", err)
	}
	return nil
}

// IsStale checks if a PID file exists but the process is not running (stale).
// Returns true if the PID file exists and the process is no longer alive.
func IsStale(path string) bool {
	pid, err := ReadPID(path)
	if err != nil {
		return false
	}
	return !isProcessRunning(pid)
}
