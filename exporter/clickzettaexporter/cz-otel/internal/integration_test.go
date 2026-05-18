// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/collector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
)

// TestE2E_ConfigSetValidateGenerate exercises the full flow:
// config set → config validate → config generate.
func TestE2E_ConfigSetValidateGenerate(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir + "/config.yaml")

	// Set all required config values.
	values := map[string]string{
		"service":         "example.clickzetta.com",
		"username":        "testuser",
		"password":        "s3cret",
		"workspace":       "ws_prod",
		"virtual_cluster": "vc_main",
		"instance":        "inst01",
	}
	for key, val := range values {
		if err := store.Set(key, val); err != nil {
			t.Fatalf("Set(%q, %q) failed: %v", key, val, err)
		}
	}

	// Load the stored data and run validation — should succeed.
	data, err := store.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	errs := config.ValidateAll(data)
	if len(errs) > 0 {
		t.Fatalf("ValidateAll should pass with all required keys set, got errors: %v", errs)
	}

	// Generate collector config — should succeed.
	yamlBytes, err := collector.Generate(data)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	yamlStr := string(yamlBytes)

	// Assert generated YAML contains all set values.
	for key, val := range values {
		expected := key + ": " + val
		if !strings.Contains(yamlStr, expected) {
			t.Errorf("generated YAML missing %q", expected)
		}
	}

	// Assert generated YAML has expected top-level sections.
	requiredSections := []string{"receivers:", "processors:", "exporters:", "service:"}
	for _, section := range requiredSections {
		if !strings.Contains(yamlStr, section) {
			t.Errorf("generated YAML missing section %q", section)
		}
	}

	// Assert default values are applied for optional fields.
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
			t.Errorf("generated YAML missing default %q", expected)
		}
	}
}

// TestE2E_IncompleteConfigValidationFails verifies that validation fails
// when only some required keys are set.
func TestE2E_IncompleteConfigValidationFails(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir + "/config.yaml")

	// Set only a subset of required keys.
	partialValues := map[string]string{
		"service":  "example.clickzetta.com",
		"username": "testuser",
	}
	for key, val := range partialValues {
		if err := store.Set(key, val); err != nil {
			t.Fatalf("Set(%q, %q) failed: %v", key, val, err)
		}
	}

	data, err := store.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Validation should fail.
	errs := config.ValidateAll(data)
	if len(errs) == 0 {
		t.Fatal("ValidateAll should fail with incomplete config, but returned no errors")
	}

	// The missing keys should be reported.
	missingKeys := []string{"password", "workspace", "virtual_cluster", "instance"}
	errStr := strings.Join(errs, " ")
	for _, key := range missingKeys {
		if !strings.Contains(errStr, key) {
			t.Errorf("validation errors should mention missing key %q, got: %v", key, errs)
		}
	}

	// Generate should also fail.
	_, genErr := collector.Generate(data)
	if genErr == nil {
		t.Fatal("Generate() should fail with incomplete config")
	}
}

// TestE2E_ConfigOverwriteAndRetrieve verifies that overwriting a key
// and then reading it back returns the new value.
func TestE2E_ConfigOverwriteAndRetrieve(t *testing.T) {
	dir := t.TempDir()
	store := config.NewStore(dir + "/config.yaml")

	// Set initial value.
	if err := store.Set("service", "old.example.com"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Overwrite with new value.
	if err := store.Set("service", "new.example.com"); err != nil {
		t.Fatalf("Set (overwrite) failed: %v", err)
	}

	got, err := store.Get("service")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != "new.example.com" {
		t.Fatalf("expected %q after overwrite, got %q", "new.example.com", got)
	}
}
