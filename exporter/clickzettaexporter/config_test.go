// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
)

func validConfig() *Config {
	cfg := createDefaultConfig().(*Config)
	cfg.Service = "api.example.clickzetta.com"
	cfg.Username = "user"
	cfg.Password = configopaque.String("pass")
	cfg.Workspace = "ws"
	cfg.VirtualCluster = "vc"
	cfg.Instance = "inst"
	return cfg
}

func TestConfig_Validate_AllRequired(t *testing.T) {
	require.NoError(t, validConfig().Validate())
}

func TestConfig_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{"no service", func(c *Config) { c.Service = "" }, "service must be specified"},
		{"no username", func(c *Config) { c.Username = "" }, "username must be specified"},
		{"no password", func(c *Config) { c.Password = "" }, "password must be specified"},
		{"no workspace", func(c *Config) { c.Workspace = "" }, "workspace must be specified"},
		{"no virtual_cluster", func(c *Config) { c.VirtualCluster = "" }, "virtual_cluster must be specified"},
		{"no instance", func(c *Config) { c.Instance = "" }, "instance must be specified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestConfig_Validate_InvalidTableNames(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
	}{
		{"semicolon injection", "otel_logs; DROP TABLE users"},
		{"dash not allowed", "otel-logs"},
		{"space not allowed", "otel logs"},
		{"leading digit", "1otel_logs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			cfg.LogsTableName = tt.tableName
			err := cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid characters")
		})
	}
}

func TestConfig_Validate_ValidTableNames(t *testing.T) {
	tests := []string{
		"otel_logs",
		"otel_logs_v2",
		"schema.otel_logs",
		"_private",
		"CamelCase",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := validConfig()
			cfg.LogsTableName = name
			require.NoError(t, cfg.Validate())
		})
	}
}

func TestConfig_DSN(t *testing.T) {
	cfg := validConfig()
	dsn := cfg.DSN()
	assert.Contains(t, dsn, "user:")
	assert.Contains(t, dsn, "@https(api.example.clickzetta.com)/public")
	assert.Contains(t, dsn, "virtualCluster=vc")
	assert.Contains(t, dsn, "workspace=ws")
	assert.Contains(t, dsn, "instance=inst")
	assert.Contains(t, dsn, "pass")
}

func TestConfig_RedactedDSN(t *testing.T) {
	cfg := validConfig()
	redacted := cfg.redactedDSN()
	assert.NotContains(t, redacted, "pass")
	assert.Contains(t, redacted, "***")
	assert.Contains(t, redacted, "user:")
	assert.Contains(t, redacted, "virtualCluster=vc")
}
