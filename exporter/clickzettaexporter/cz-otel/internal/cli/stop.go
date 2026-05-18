// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/daemon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the OpenTelemetry Collector daemon",
	Long:  `Stops the running OpenTelemetry Collector daemon process.`,
	Args:  cobra.NoArgs,
	RunE:  runStop,
}

func init() {
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, _ []string) error {
	pidPath := paths.PIDFilePath()

	if !daemon.IsRunning(pidPath) {
		// Check for stale PID file and clean up.
		if daemon.IsStale(pidPath) {
			_ = daemon.RemovePID(pidPath)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No collector is currently running.")
		return nil
	}

	if err := daemon.Stop(pidPath); err != nil {
		return fmt.Errorf("stopping collector: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Collector stopped.")
	return nil
}
