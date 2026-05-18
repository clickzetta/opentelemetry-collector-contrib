// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/daemon"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the OpenTelemetry Collector daemon",
	Long:  `Reports whether the OpenTelemetry Collector daemon is running or stopped.`,
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, _ []string) error {
	pidPath := paths.PIDFilePath()

	info, running, err := daemon.Status(pidPath)
	if err != nil {
		return fmt.Errorf("checking collector status: %w", err)
	}

	if running {
		fmt.Fprintf(cmd.OutOrStdout(), "Collector is running (PID: %d)\n", info.PID)
		fmt.Fprintf(cmd.OutOrStdout(), "Logs: %s\n", paths.LogFilePath())
		return nil
	}

	// PID file exists but process is not running (stale).
	if info != nil {
		_ = daemon.RemovePID(pidPath)
		fmt.Fprintln(cmd.OutOrStdout(), "Collector is not running (stale PID file cleaned up).")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'cz-otel start' to start the collector.")
		return nil
	}

	// No PID file at all — check if the file simply doesn't exist.
	if _, statErr := os.Stat(pidPath); os.IsNotExist(statErr) {
		fmt.Fprintln(cmd.OutOrStdout(), "Collector is not running.")
		return nil
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Collector is not running.")
	return nil
}
