// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"
)

// counter is used to generate unique temp directories within rapid tests.
var counter atomic.Int64

// validKeyGen generates a random valid config key (excluding "password" for round-trip tests).
func validKeyGen() *rapid.Generator[string] {
	nonPasswordKeys := []string{
		"service", "username", "workspace", "virtual_cluster",
		"instance", "schema", "protocol",
		"logs_table_name", "traces_table_name", "metrics_table_name",
		"create_schema",
	}
	return rapid.SampledFrom(nonPasswordKeys)
}

// safeValueGen generates non-empty string values suitable for config values.
func safeValueGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9_.\-]{1,50}`)
}

// newTempStore creates a Store backed by a temporary file for testing.
func newTempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(filepath.Join(dir, "config.yaml"))
}

// newTempStoreInDir creates a Store in the given base directory with a unique subdirectory.
func newTempStoreInDir(baseDir string) *Store {
	id := counter.Add(1)
	dir := filepath.Join(baseDir, fmt.Sprintf("run_%d", id))
	_ = os.MkdirAll(dir, 0700)
	return NewStore(filepath.Join(dir, "config.yaml"))
}

// Feature: cz-otel-cli, Property 1: Config Store Round-Trip
// For any valid configuration key-value pair, storing it via Set and retrieving
// it via Get shall return the original value.
// **Validates: Requirements 3.1, 3.3**
func TestProperty1_RoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		store := newTempStoreInDir(baseDir)

		key := validKeyGen().Draw(t, "key")
		value := safeValueGen().Draw(t, "value")

		err := store.Set(key, value)
		if err != nil {
			t.Fatalf("Set(%q, %q) failed: %v", key, value, err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", key, err)
		}

		if got != value {
			t.Fatalf("Round-trip failed: Set(%q, %q) then Get(%q) = %q", key, value, key, got)
		}
	})
}

// Feature: cz-otel-cli, Property 8: Config Overwrite Idempotence
// Setting the same key to the same value multiple times produces the same
// result as setting it once.
// **Validates: Requirements 3.2**
func TestProperty8_OverwriteIdempotence(t *testing.T) {
	baseDir := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		store := newTempStoreInDir(baseDir)

		key := validKeyGen().Draw(t, "key")
		value := safeValueGen().Draw(t, "value")
		times := rapid.IntRange(2, 10).Draw(t, "times")

		// Set the value multiple times.
		for i := 0; i < times; i++ {
			err := store.Set(key, value)
			if err != nil {
				t.Fatalf("Set(%q, %q) iteration %d failed: %v", key, value, i, err)
			}
		}

		// Verify the stored value is the same as if set once.
		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", key, err)
		}
		if got != value {
			t.Fatalf("Idempotence failed: after %d sets of (%q, %q), Get returned %q", times, key, value, got)
		}

		// Verify the file contains exactly one entry for this key.
		cfg, err := store.Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}
		if cfg[key] != value {
			t.Fatalf("Load() returned %q for key %q, expected %q", cfg[key], key, value)
		}
	})
}

// Feature: cz-otel-cli, Property 9: Default Values for Optional Fields
// For optional fields not explicitly set, Get returns the defined default value.
// **Validates: Requirements 4.4**
func TestProperty9_DefaultValues(t *testing.T) {
	baseDir := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		store := newTempStoreInDir(baseDir)

		// Pick a key that has a default value.
		optionalKeys := []string{"schema", "protocol", "logs_table_name", "traces_table_name", "metrics_table_name", "create_schema"}
		key := rapid.SampledFrom(optionalKeys).Draw(t, "key")

		// Without setting anything, Get should return the default.
		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get(%q) failed: %v", key, err)
		}

		expected := DefaultValue(key)
		if got != expected {
			t.Fatalf("Default value for %q: got %q, expected %q", key, got, expected)
		}
	})
}

// Feature: cz-otel-cli, Property 7: Password Redaction in List Output
// List() never contains the actual password value — it is always replaced with "********".
// **Validates: Requirements 3.5**
func TestProperty7_PasswordRedaction(t *testing.T) {
	baseDir := t.TempDir()
	rapid.Check(t, func(t *rapid.T) {
		store := newTempStoreInDir(baseDir)

		password := safeValueGen().Draw(t, "password")

		err := store.Set("password", password)
		if err != nil {
			t.Fatalf("Set password failed: %v", err)
		}

		listed, err := store.List()
		if err != nil {
			t.Fatalf("List() failed: %v", err)
		}

		if listed["password"] == password {
			t.Fatalf("List() exposed actual password %q", password)
		}
		if listed["password"] != redactedPassword {
			t.Fatalf("List() password = %q, expected %q", listed["password"], redactedPassword)
		}
	})
}

// TestSaveFilePermissions verifies that Save creates files with 0600 permissions on Unix.
func TestSaveFilePermissions(t *testing.T) {
	store := newTempStore(t)

	err := store.Set("service", "test.example.com")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	info, err := os.Stat(store.path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// On Unix, verify 0600 permissions.
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("Expected file permissions 0600, got %04o", perm)
	}
}
