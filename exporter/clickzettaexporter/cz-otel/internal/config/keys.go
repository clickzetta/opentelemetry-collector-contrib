// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package config

// ValidKeys is the complete set of recognized configuration keys.
var ValidKeys = []string{
	"service",
	"username",
	"password",
	"workspace",
	"virtual_cluster",
	"instance",
	"schema",
	"protocol",
	"logs_table_name",
	"traces_table_name",
	"metrics_table_name",
	"create_schema",
}

// RequiredKeys is the subset of keys that must be present for a valid configuration.
var RequiredKeys = []string{
	"service",
	"username",
	"password",
	"workspace",
	"virtual_cluster",
	"instance",
}

// DefaultValues provides default values for optional configuration keys.
var DefaultValues = map[string]string{
	"schema":             "public",
	"protocol":           "https",
	"logs_table_name":    "otel_logs",
	"traces_table_name":  "otel_traces",
	"metrics_table_name": "otel_metrics",
	"create_schema":      "true",
}

// validKeySet is a pre-computed set for O(1) lookup.
var validKeySet map[string]bool

// requiredKeySet is a pre-computed set for O(1) lookup.
var requiredKeySet map[string]bool

func init() {
	validKeySet = make(map[string]bool, len(ValidKeys))
	for _, k := range ValidKeys {
		validKeySet[k] = true
	}
	requiredKeySet = make(map[string]bool, len(RequiredKeys))
	for _, k := range RequiredKeys {
		requiredKeySet[k] = true
	}
}

// IsValidKey returns true if the given key is a recognized configuration key.
func IsValidKey(key string) bool {
	return validKeySet[key]
}

// IsRequiredKey returns true if the given key is a required configuration key.
func IsRequiredKey(key string) bool {
	return requiredKeySet[key]
}

// DefaultValue returns the default value for the given key, or an empty string
// if no default is defined.
func DefaultValue(key string) string {
	return DefaultValues[key]
}
