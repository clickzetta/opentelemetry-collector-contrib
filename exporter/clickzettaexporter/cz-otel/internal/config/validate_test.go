// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// Feature: cz-otel-cli, Property 2: Config Key Validation Completeness
// For any string, it is either in the set of valid keys (accepted) or not (rejected).
// No key outside the defined set is accepted, and no key inside the set is rejected.
// **Validates: Requirements 3.7, 3.8**
func TestProperty2_KeyValidationCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.String().Draw(t, "key")

		isValid := IsValidKey(key)

		// Check against the ground truth set.
		inSet := false
		for _, k := range ValidKeys {
			if k == key {
				inSet = true
				break
			}
		}

		if isValid != inSet {
			t.Fatalf("IsValidKey(%q) = %v, but key in ValidKeys set = %v", key, isValid, inSet)
		}
	})
}

// Feature: cz-otel-cli, Property 4: Validation Rejects Incomplete Configs
// For any config missing at least one required key, validation fails and the
// error message contains exactly the set of missing keys.
// **Validates: Requirements 10.1, 10.3**
func TestProperty4_ValidationRejectsIncompleteConfigs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Build a config with a random subset of required keys present.
		cfg := make(map[string]string)

		// Decide which required keys to include (at least one must be missing).
		includeMask := rapid.IntRange(0, (1<<len(RequiredKeys))-2).Draw(t, "includeMask")

		var missing []string
		for i, key := range RequiredKeys {
			if includeMask&(1<<i) != 0 {
				cfg[key] = "some_value"
			} else {
				missing = append(missing, key)
			}
		}

		// Ensure at least one key is missing.
		if len(missing) == 0 {
			t.Skip("no missing keys in this draw")
		}

		err := Validate(cfg)
		if err == nil {
			t.Fatalf("Validate() should have failed with missing keys %v, but returned nil", missing)
		}

		// Verify the error mentions each missing key.
		errMsg := err.Error()
		for _, key := range missing {
			if !containsWord(errMsg, key) {
				t.Fatalf("Validate() error %q does not mention missing key %q", errMsg, key)
			}
		}
	})
}

// Feature: cz-otel-cli, Property 5: Table Name Validation Pattern
// For any string, it passes table name validation if and only if it matches
// the pattern ^[a-zA-Z_][a-zA-Z0-9_.]*$
// **Validates: Requirements 10.4**
func TestProperty5_TableNameValidationPattern(t *testing.T) {
	pattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)

	rapid.Check(t, func(t *rapid.T) {
		name := rapid.String().Draw(t, "name")

		got := ValidateTableName(name)
		expected := pattern.MatchString(name)

		if got != expected {
			t.Fatalf("ValidateTableName(%q) = %v, but regex match = %v", name, got, expected)
		}
	})
}

// containsWord checks if a string contains a given substring.
func containsWord(s, word string) bool {
	return len(s) > 0 && len(word) > 0 && contains(s, word)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
