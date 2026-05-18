// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
)

// promptFields defines the order and metadata for interactive prompts.
var promptFields = []struct {
	Key      string
	Label    string
	Required bool
	Secret   bool
}{
	{Key: "service", Label: "Service endpoint", Required: true},
	{Key: "username", Label: "Username", Required: true},
	{Key: "password", Label: "Password", Required: true, Secret: true},
	{Key: "workspace", Label: "Workspace", Required: true},
	{Key: "virtual_cluster", Label: "Virtual cluster", Required: true},
	{Key: "instance", Label: "Instance", Required: true},
	{Key: "schema", Label: "Schema", Required: false},
	{Key: "protocol", Label: "Protocol", Required: false},
	{Key: "logs_table_name", Label: "Logs table name", Required: false},
	{Key: "traces_table_name", Label: "Traces table name", Required: false},
	{Key: "metrics_table_name", Label: "Metrics table name", Required: false},
	{Key: "create_schema", Label: "Create schema (true/false)", Required: false},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive configuration wizard",
	Long:  `Interactively prompts for each configuration field and saves the values.`,
	Args:  cobra.NoArgs,
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(configInitCmd)
}

func runConfigInit(cmd *cobra.Command, _ []string) error {
	store := config.NewStore("")
	existing, err := store.Load()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	result := make(map[string]string)

	fmt.Fprintln(cmd.OutOrStdout(), "ClickZetta OpenTelemetry Collector Configuration")
	fmt.Fprintln(cmd.OutOrStdout(), "Press Enter to keep the existing/default value shown in brackets.")
	fmt.Fprintln(cmd.OutOrStdout())

	for _, field := range promptFields {
		// Determine the current/default value to show.
		currentValue := existing[field.Key]
		if currentValue == "" {
			currentValue = config.DefaultValue(field.Key)
		}

		var value string
		if field.Secret {
			value, err = promptSecret(cmd, field.Label, currentValue)
			if err != nil {
				return err
			}
		} else {
			value = promptField(cmd, scanner, field.Label, currentValue)
		}

		// If user entered nothing, keep the current/default value.
		if value == "" {
			value = currentValue
		}

		if value != "" {
			result[field.Key] = value
		}
	}

	if err := store.Save(result); err != nil {
		return fmt.Errorf("saving configuration: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Configuration saved successfully.")
	fmt.Fprintln(cmd.OutOrStdout(), "Run 'cz-otel config validate' to verify, or 'cz-otel start' to launch the collector.")
	return nil
}

// promptField displays a prompt and reads a line of input.
func promptField(cmd *cobra.Command, scanner *bufio.Scanner, label, defaultValue string) string {
	if defaultValue != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s: ", label)
	}
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// promptSecret reads a password without echoing characters to the terminal.
func promptSecret(cmd *cobra.Command, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s [********]: ", label)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s: ", label)
	}

	// Try to read password with terminal raw mode for masking.
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		password, err := term.ReadPassword(fd)
		fmt.Fprintln(cmd.OutOrStdout()) // Print newline after hidden input.
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		return strings.TrimSpace(string(password)), nil
	}

	// Fallback for non-terminal (e.g., piped input): read normally.
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	return "", nil
}
