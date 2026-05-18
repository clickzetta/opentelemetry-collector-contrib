// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/collector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/daemon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var foreground bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the OpenTelemetry Collector daemon",
	Long: `Starts the OpenTelemetry Collector as a background daemon process.
The collector binary must be installed at the expected path.
Configuration is validated and the collector config is generated before starting.`,
	Args: cobra.NoArgs,
	RunE: runStart,
}

func init() {
	startCmd.Flags().BoolVar(&foreground, "foreground", false, "run the collector in the foreground")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, _ []string) error {
	// Check if collector binary exists.
	binaryPath := paths.CollectorBinaryPath()
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("collector binary not found at %s — please re-run the install script", binaryPath)
	}

	// Load and validate config.
	store := config.NewStore("")
	data, err := store.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	errors := config.ValidateAll(data)
	if len(errors) > 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), "Configuration validation failed:")
		for _, e := range errors {
			fmt.Fprintf(cmd.ErrOrStderr(), "  - %s\n", e)
		}
		return fmt.Errorf("fix configuration errors before starting (use 'cz-otel config set' or 'cz-otel config init')")
	}

	// Generate collector config.
	yamlBytes, err := collector.Generate(data)
	if err != nil {
		return fmt.Errorf("generating collector config: %w", err)
	}

	configPath := paths.CollectorConfigPath()
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(configPath, yamlBytes, 0600); err != nil {
		return fmt.Errorf("writing collector config: %w", err)
	}

	// Foreground mode: run the collector attached to the terminal.
	if foreground {
		return runForeground(cmd, binaryPath, configPath)
	}

	// Check if already running.
	pidPath := paths.PIDFilePath()
	if daemon.IsRunning(pidPath) {
		pid, _ := daemon.ReadPID(pidPath)
		fmt.Fprintf(cmd.OutOrStdout(), "Collector is already running (PID: %d)\n", pid)
		return nil
	}

	// Clean up stale PID file if present.
	if daemon.IsStale(pidPath) {
		_ = daemon.RemovePID(pidPath)
	}

	// Start daemon.
	logPath := paths.LogFilePath()
	info, err := daemon.Start(binaryPath, configPath, logPath, pidPath)
	if err != nil {
		return fmt.Errorf("starting collector daemon: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Collector started (PID: %d)\n", info.PID)
	fmt.Fprintf(cmd.OutOrStdout(), "Logs: %s\n", logPath)
	return nil
}

// runForeground runs the collector in the foreground, attached to the current terminal.
func runForeground(cmd *cobra.Command, binaryPath, configPath string) error {
	proc := exec.Command(binaryPath, "--config="+configPath)
	proc.Stdout = cmd.OutOrStdout()
	proc.Stderr = cmd.ErrOrStderr()

	// Forward interrupt signals to the child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting collector in foreground: %w", err)
	}

	go func() {
		sig := <-sigCh
		if proc.Process != nil {
			_ = proc.Process.Signal(sig)
		}
	}()

	return proc.Wait()
}
