// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"regexp"
	"strings"
)

// tableNamePattern defines the valid pattern for table names.
var tableNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

// Validate checks that all required keys are present in the configuration data.
// Returns an error listing missing keys, or nil if all required keys are present.
func Validate(data map[string]string) error {
	var missing []string
	for _, key := range RequiredKeys {
		if val, ok := data[key]; !ok || val == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config keys: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ValidateTableName returns true if the given name matches the valid table name
// pattern: starts with a letter or underscore, followed by letters, digits,
// underscores, or dots.
func ValidateTableName(name string) bool {
	return tableNamePattern.MatchString(name)
}

// ValidateAll performs comprehensive validation and returns a list of all errors found.
// It checks for missing required keys and invalid table names.
func ValidateAll(data map[string]string) []string {
	var errors []string

	// Check required keys.
	for _, key := range RequiredKeys {
		if val, ok := data[key]; !ok || val == "" {
			errors = append(errors, fmt.Sprintf("missing required key: %s", key))
		}
	}

	// Check table name patterns.
	tableKeys := []string{"logs_table_name", "traces_table_name", "metrics_table_name"}
	for _, key := range tableKeys {
		if val, ok := data[key]; ok && val != "" {
			if !ValidateTableName(val) {
				errors = append(errors, fmt.Sprintf("invalid table name for %s: %q (must match %s)", key, val, tableNamePattern.String()))
			}
		}
	}

	return errors
}
