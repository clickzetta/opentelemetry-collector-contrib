// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/collector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/daemon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the OpenTelemetry Collector daemon",
	Long: `Restarts the OpenTelemetry Collector daemon. If the collector is running,
it will be stopped first. Configuration is re-validated and the collector config
is regenerated before starting.`,
	Args: cobra.NoArgs,
	RunE: runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, _ []string) error {
	pidPath := paths.PIDFilePath()

	// Stop if currently running.
	if daemon.IsRunning(pidPath) {
		if err := daemon.Stop(pidPath); err != nil {
			return fmt.Errorf("stopping collector for restart: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Collector stopped.")
	} else if daemon.IsStale(pidPath) {
		// Clean up stale PID file.
		_ = daemon.RemovePID(pidPath)
	}

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
		return fmt.Errorf("fix configuration errors before restarting (use 'cz-otel config set' or 'cz-otel config init')")
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

	// Start daemon.
	logPath := paths.LogFilePath()
	info, err := daemon.Start(binaryPath, configPath, logPath, pidPath)
	if err != nil {
		return fmt.Errorf("starting collector daemon: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Collector restarted (PID: %d)\n", info.PID)
	fmt.Fprintf(cmd.OutOrStdout(), "Logs: %s\n", logPath)
	return nil
}
