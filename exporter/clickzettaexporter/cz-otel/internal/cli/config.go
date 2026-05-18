// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cz-otel configuration",
	Long:  `Manage cz-otel configuration values used to generate the collector config.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := config.NewStore("")
		key, value := args[0], args[1]
		if err := store.Set(key, value); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := config.NewStore("")
		value, err := store.Get(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := config.NewStore("")
		values, err := store.List()
		if err != nil {
			return err
		}
		if len(values) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No configuration values set.")
			return nil
		}
		// Sort keys for consistent output.
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", k, values[k])
		}
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the current configuration",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store := config.NewStore("")
		data, err := store.Load()
		if err != nil {
			return err
		}
		errors := config.ValidateAll(data)
		if len(errors) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "Configuration is valid.")
			return nil
		}
		for _, e := range errors {
			fmt.Fprintln(cmd.ErrOrStderr(), e)
		}
		os.Exit(1)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
