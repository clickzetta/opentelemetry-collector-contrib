// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// startProcess starts the collector process on Windows.
// It redirects stdout and stderr to the log file.
func startProcess(binaryPath, configPath, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("opening log file: %w", err)
	}

	cmd := exec.Command(binaryPath, "--config="+configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("starting collector process: %w", err)
	}

	// Close the log file in the parent process; the child has its own file descriptors.
	logFile.Close()

	return cmd.Process.Pid, nil
}

// stopProcess finds the process by PID and kills it on Windows.
func stopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := process.Kill(); err != nil {
		if err == os.ErrProcessDone {
			return nil
		}
		return fmt.Errorf("killing process %d: %w", pid, err)
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is alive on Windows.
// On Windows, os.FindProcess always succeeds, so we attempt to signal the process
// to determine if it's actually running.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, sending os.Kill and checking the error is a way to probe.
	// However, that would kill the process. Instead, we use a zero-signal equivalent.
	// Go on Windows maps Signal(os.Interrupt) to GenerateConsoleCtrlEvent which may fail
	// for processes we don't own. The safest approach is to try Signal with a nil-like check.
	// We rely on the fact that process.Signal returns os.ErrProcessDone if the process exited.
	err = process.Signal(os.Signal(syscall.Signal(0)))
	if err == nil {
		return true
	}
	// If the error indicates the process is done, it's not running.
	if err == os.ErrProcessDone {
		return false
	}
	// Other errors (e.g., access denied) may indicate the process exists.
	return true
}
