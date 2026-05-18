// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config defines configuration for the ClickZetta exporter.
type Config struct {
	TimeoutSettings           exporterhelper.TimeoutConfig `mapstructure:",squash"`
	configretry.BackOffConfig `mapstructure:"retry_on_failure"`
	QueueSettings             configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`

	// Service is the ClickZetta service endpoint URL.
	Service string `mapstructure:"service"`
	// Username is the authentication username.
	Username string `mapstructure:"username"`
	// Password is the authentication password.
	Password configopaque.String `mapstructure:"password"`
	// Workspace is the ClickZetta workspace name.
	Workspace string `mapstructure:"workspace"`
	// VirtualCluster is the ClickZetta virtual cluster name.
	VirtualCluster string `mapstructure:"virtual_cluster"`
	// Instance is the ClickZetta instance name.
	Instance string `mapstructure:"instance"`
	// Schema is the ClickZetta schema name. Default is "public".
	Schema string `mapstructure:"schema"`
	// Protocol is the connection protocol. Default is "https".
	Protocol string `mapstructure:"protocol"`

	// LogsTableName is the table name for logs. Default is "otel_logs".
	LogsTableName string `mapstructure:"logs_table_name"`
	// TracesTableName is the table name for traces. Default is "otel_traces".
	TracesTableName string `mapstructure:"traces_table_name"`
	// MetricsTableName is the base table name prefix for metrics. Default is "otel_metrics".
	MetricsTableName string `mapstructure:"metrics_table_name"`
	// CreateSchema if true will run DDL for creating tables. Default is true.
	CreateSchema bool `mapstructure:"create_schema"`

	// APIKeyServiceURL is the base URL of the Key Service.
	// When set, enables API key gateway mode.
	APIKeyServiceURL string `mapstructure:"apikey_service_url"`
	// APIKeyHeader is the metadata key to extract the API key from.
	// Default: "x-api-key"
	APIKeyHeader string `mapstructure:"apikey_header"`
	// RouterMode selects the router implementation.
	// Valid values: "real", "mock". Default: "real"
	RouterMode string `mapstructure:"router_mode"`
	// CacheTTL is the duration to cache Key Service responses.
	// Default: 5m
	CacheTTL time.Duration `mapstructure:"cache_ttl"`
}

var (
	errNoService        = errors.New("service must be specified")
	errNoUsername       = errors.New("username must be specified")
	errNoPassword       = errors.New("password must be specified")
	errNoWorkspace      = errors.New("workspace must be specified")
	errNoVirtualCluster = errors.New("virtual_cluster must be specified")
	errNoInstance       = errors.New("instance must be specified")
	errInvalidRouterMode = errors.New(`router_mode must be "real" or "mock"`)

	validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_.]*$`)
)

func createDefaultConfig() component.Config {
	timeout := exporterhelper.NewDefaultTimeoutConfig()
	timeout.Timeout = 10 * time.Second
	return &Config{
		TimeoutSettings: timeout,
		QueueSettings:   configoptional.Some(exporterhelper.NewDefaultQueueConfig()),
		BackOffConfig:   configretry.NewDefaultBackOffConfig(),
		Schema:          "public",
		Protocol:        "https",
		LogsTableName:   "otel_logs",
		TracesTableName: "otel_traces",
		MetricsTableName: "otel_metrics",
		CreateSchema:    true,
		APIKeyHeader:    "x-api-key",
		RouterMode:      "real",
		CacheTTL:        5 * time.Minute,
	}
}

// Validate checks the configuration for required fields.
func (cfg *Config) Validate() error {
	var err error

	if cfg.isSingleTenantMode() {
		// Single-tenant mode requires all connection fields.
		if cfg.Service == "" {
			err = errors.Join(err, errNoService)
		}
		if cfg.Username == "" {
			err = errors.Join(err, errNoUsername)
		}
		if string(cfg.Password) == "" {
			err = errors.Join(err, errNoPassword)
		}
		if cfg.Workspace == "" {
			err = errors.Join(err, errNoWorkspace)
		}
		if cfg.VirtualCluster == "" {
			err = errors.Join(err, errNoVirtualCluster)
		}
		if cfg.Instance == "" {
			err = errors.Join(err, errNoInstance)
		}
	}

	if cfg.isGatewayMode() {
		// Gateway mode requires a valid router_mode.
		if cfg.RouterMode != "" && cfg.RouterMode != "real" && cfg.RouterMode != "mock" {
			err = errors.Join(err, errInvalidRouterMode)
		}
	}

	for _, name := range []string{cfg.LogsTableName, cfg.TracesTableName, cfg.MetricsTableName} {
		if name != "" && !validIdentifier.MatchString(name) {
			err = errors.Join(err, fmt.Errorf("table name %q contains invalid characters; must match [a-zA-Z_][a-zA-Z0-9_.]*", name))
		}
	}
	return err
}

// isGatewayMode returns true when the exporter is configured for API key gateway mode.
func (cfg *Config) isGatewayMode() bool {
	return cfg.APIKeyServiceURL != ""
}

// isSingleTenantMode returns true when the exporter is configured for single-tenant mode.
func (cfg *Config) isSingleTenantMode() bool {
	return !cfg.isGatewayMode()
}

// DSN builds the ClickZetta DSN string from config.
func (cfg *Config) DSN() string {
	// Format: user:pwd@protocol(service)/schema?virtualCluster=vc&workspace=ws&instance=inst
	return cfg.Username + ":" + string(cfg.Password) + "@" + cfg.Protocol + "(" + cfg.Service + ")/" + cfg.Schema +
		"?virtualCluster=" + cfg.VirtualCluster + "&workspace=" + cfg.Workspace + "&instance=" + cfg.Instance
}

// redactedDSN returns a DSN with the password replaced by *** for safe logging.
func (cfg *Config) redactedDSN() string {
	return cfg.Username + ":***@" + cfg.Protocol + "(" + cfg.Service + ")/" + cfg.Schema +
		"?virtualCluster=" + cfg.VirtualCluster + "&workspace=" + cfg.Workspace + "&instance=" + cfg.Instance
}
