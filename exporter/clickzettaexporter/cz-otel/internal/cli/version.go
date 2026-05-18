// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information for the CLI and collector",
	Long:  `Prints the CLI version, collector version, installation status, and platform.`,
	Args:  cobra.NoArgs,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, _ []string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "cz-otel version %s\n", cliVersion)

	installStatus := "not installed"
	if _, err := os.Stat(paths.CollectorBinaryPath()); err == nil {
		installStatus = "installed"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Collector: %s (%s)\n", cliCollectorVersion, installStatus)
	fmt.Fprintf(cmd.OutOrStdout(), "Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return nil
}
