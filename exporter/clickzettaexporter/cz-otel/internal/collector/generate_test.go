// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
)

// Feature: cz-otel-cli, Property 3: Generated Config Contains All Store Values
//
// For any complete and valid Config_Store, the generated Collector_Config YAML
// contains every non-default value from the store in the clickzetta exporter section.
//
// **Validates: Requirements 5.1, 5.4**

// genAlphaNum generates a non-empty alphanumeric string suitable for config values.
func genAlphaNum() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{2,20}`)
}

// genTableName generates a valid table name matching the pattern [a-zA-Z_][a-zA-Z0-9_.]*
func genTableName() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{2,15}`)
}

// genServiceURL generates a plausible service URL string.
func genServiceURL() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z]{3,10}\.[a-z]{3,10}\.(com|io|net)`)
}

// genCompleteConfig generates a complete valid configuration map with all required
// and optional fields populated with non-default values.
func genCompleteConfig(t *rapid.T) map[string]string {
	data := map[string]string{
		"service":            genServiceURL().Draw(t, "service"),
		"username":           genAlphaNum().Draw(t, "username"),
		"password":           genAlphaNum().Draw(t, "password"),
		"workspace":          genAlphaNum().Draw(t, "workspace"),
		"virtual_cluster":    genAlphaNum().Draw(t, "virtual_cluster"),
		"instance":           genAlphaNum().Draw(t, "instance"),
		"schema":             genAlphaNum().Draw(t, "schema"),
		"create_schema":      rapid.SampledFrom([]string{"true", "false"}).Draw(t, "create_schema"),
		"logs_table_name":    genTableName().Draw(t, "logs_table_name"),
		"traces_table_name":  genTableName().Draw(t, "traces_table_name"),
		"metrics_table_name": genTableName().Draw(t, "metrics_table_name"),
	}
	return data
}

func TestProperty3_GeneratedConfigContainsAllStoreValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		data := genCompleteConfig(t)

		yamlBytes, err := Generate(data)
		if err != nil {
			t.Fatalf("Generate returned error for valid config: %v", err)
		}

		yamlStr := string(yamlBytes)

		// The generated YAML must contain every non-default value from the store
		// in the clickzetta exporter section.
		exporterIdx := strings.Index(yamlStr, "exporters:")
		serviceIdx := strings.Index(yamlStr, "service:\n  pipelines:")
		if exporterIdx < 0 || serviceIdx < 0 {
			t.Fatal("generated YAML missing exporters or service section")
		}
		exporterSection := yamlStr[exporterIdx:serviceIdx]

		// Check that all exporter keys appear with their values in the exporter section.
		exporterKeys := []string{
			"service", "username", "password", "workspace",
			"virtual_cluster", "instance", "schema", "create_schema",
			"logs_table_name", "traces_table_name", "metrics_table_name",
		}
		for _, key := range exporterKeys {
			val := data[key]
			expectedEntry := key + ": " + val
			if !strings.Contains(exporterSection, expectedEntry) {
				t.Errorf("exporter section missing %q (expected %q)", key, expectedEntry)
			}
		}
	})
}

func TestGenerate_ErrorOnMissingRequiredKeys(t *testing.T) {
	// Test that Generate returns an error when required keys are missing.
	for _, requiredKey := range config.RequiredKeys {
		t.Run("missing_"+requiredKey, func(t *testing.T) {
			// Build a complete config, then remove one required key.
			data := map[string]string{
				"service":         "test.example.com",
				"username":        "testuser",
				"password":        "testpass",
				"workspace":       "ws1",
				"virtual_cluster": "vc1",
				"instance":        "inst1",
			}
			delete(data, requiredKey)

			_, err := Generate(data)
			if err == nil {
				t.Errorf("expected error when %q is missing, got nil", requiredKey)
			}
			if !strings.Contains(err.Error(), requiredKey) {
				t.Errorf("error message should mention missing key %q, got: %v", requiredKey, err)
			}
		})
	}
}

func TestGenerate_AppliesDefaults(t *testing.T) {
	// Provide only required keys and verify defaults are applied.
	data := map[string]string{
		"service":         "test.example.com",
		"username":        "testuser",
		"password":        "testpass",
		"workspace":       "ws1",
		"virtual_cluster": "vc1",
		"instance":        "inst1",
	}

	yamlBytes, err := Generate(data)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}

	yamlStr := string(yamlBytes)

	// Verify defaults are present in the output.
	defaults := map[string]string{
		"schema":             "public",
		"create_schema":      "true",
		"logs_table_name":    "otel_logs",
		"traces_table_name":  "otel_traces",
		"metrics_table_name": "otel_metrics",
	}
	for key, val := range defaults {
		expected := key + ": " + val
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("expected default %q in output, not found", expected)
		}
	}
}
