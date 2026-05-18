// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cliVersion          = "dev"
	cliCollectorVersion = "unknown"
)

// SetVersionInfo sets the version information for the CLI.
func SetVersionInfo(version, collectorVersion string) {
	cliVersion = version
	cliCollectorVersion = collectorVersion
}

var rootCmd = &cobra.Command{
	Use:   "cz-otel",
	Short: "CLI tool for managing the ClickZetta OpenTelemetry Collector",
	Long: `cz-otel is a command-line tool that wraps the OpenTelemetry Collector
with the ClickZetta exporter plugin. It provides commands for managing
configuration and controlling the collector daemon lifecycle.`,
	Version: cliVersion,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("cz-otel version %s (collector: %s)\n", cliVersion, cliCollectorVersion))
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
