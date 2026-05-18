// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// startProcess starts the collector as a detached process on Unix systems.
// It redirects stdout and stderr to the log file and detaches from the terminal.
func startProcess(binaryPath, configPath, logPath string) (int, error) {
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("opening log file: %w", err)
	}

	cmd := exec.Command(binaryPath, "--config="+configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return 0, fmt.Errorf("starting collector process: %w", err)
	}

	// Close the log file in the parent process; the child has its own file descriptors.
	logFile.Close()

	return cmd.Process.Pid, nil
}

// stopProcess sends SIGTERM to the process, waits up to 10 seconds, then sends SIGKILL.
func stopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	// Send SIGTERM for graceful shutdown.
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process may already be gone.
		if err == os.ErrProcessDone {
			return nil
		}
		return fmt.Errorf("sending SIGTERM to process %d: %w", pid, err)
	}

	// Wait up to 10 seconds for the process to exit.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Process still running after timeout; send SIGKILL.
	if err := process.Signal(syscall.SIGKILL); err != nil {
		if err == os.ErrProcessDone {
			return nil
		}
		return fmt.Errorf("sending SIGKILL to process %d: %w", pid, err)
	}

	return nil
}

// isProcessRunning checks if a process with the given PID exists using signal 0.
func isProcessRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}
