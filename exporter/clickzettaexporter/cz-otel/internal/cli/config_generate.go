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
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/paths"
)

var generateOutput string

var configGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate the collector configuration YAML",
	Long:  `Generates the OpenTelemetry Collector configuration YAML from stored config values.`,
	Args:  cobra.NoArgs,
	RunE:  runConfigGenerate,
}

func init() {
	configGenerateCmd.Flags().StringVarP(&generateOutput, "output", "o", "", "output file path (default: ~/.cz-otel/otel-collector-config.yaml)")
	configCmd.AddCommand(configGenerateCmd)
}

func runConfigGenerate(cmd *cobra.Command, _ []string) error {
	store := config.NewStore("")
	data, err := store.Load()
	if err != nil {
		return err
	}

	yamlBytes, err := collector.Generate(data)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}

	outputPath := generateOutput
	if outputPath == "" {
		outputPath = paths.CollectorConfigPath()
	}

	// Ensure the output directory exists.
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, yamlBytes, 0600); err != nil {
		return fmt.Errorf("writing collector config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Collector configuration written to %s\n", outputPath)
	return nil
}
