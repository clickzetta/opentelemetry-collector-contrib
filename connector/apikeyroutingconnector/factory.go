// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:generate make mdatagen

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector/internal/metadata"
)

// NewFactory returns a ConnectorFactory for the API Key Routing Connector.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		connector.WithTracesToTraces(createTracesToTraces, metadata.TracesToTracesStability),
		connector.WithMetricsToMetrics(createMetricsToMetrics, metadata.MetricsToMetricsStability),
		connector.WithLogsToLogs(createLogsToLogs, metadata.LogsToLogsStability),
	)
}

// createDefaultConfig creates the default configuration.
func createDefaultConfig() component.Config {
	return &Config{
		KeyHeader:           "x-api-key",
		CacheTTL:            5 * time.Minute,
		KeyServiceTimeout:   5 * time.Second,
		DefaultExporterType: "clickzetta",
		ExporterIdleTimeout: 30 * time.Minute,
		BatchSize:           0, // disabled by default
		FlushInterval:       5 * time.Second,
	}
}

// createTracesToTraces creates a traces to traces connector based on provided config.
func createTracesToTraces(
	_ context.Context,
	set connector.Settings,
	cfg component.Config,
	traces consumer.Traces,
) (connector.Traces, error) {
	c := cfg.(*Config)
	return &tracesConnector{
		cfg:             c,
		logger:          set.Logger,
		defaultConsumer: traces,
		settings: exporter.Settings{
			ID:                set.ID,
			TelemetrySettings: set.TelemetrySettings,
			BuildInfo:         set.BuildInfo,
		},
	}, nil
}

// createMetricsToMetrics creates a metrics to metrics connector based on provided config.
func createMetricsToMetrics(
	_ context.Context,
	set connector.Settings,
	cfg component.Config,
	metrics consumer.Metrics,
) (connector.Metrics, error) {
	c := cfg.(*Config)
	return &metricsConnector{
		cfg:             c,
		logger:          set.Logger,
		defaultConsumer: metrics,
		settings: exporter.Settings{
			ID:                set.ID,
			TelemetrySettings: set.TelemetrySettings,
			BuildInfo:         set.BuildInfo,
		},
	}, nil
}

// createLogsToLogs creates a logs to logs connector based on provided config.
func createLogsToLogs(
	_ context.Context,
	set connector.Settings,
	cfg component.Config,
	logs consumer.Logs,
) (connector.Logs, error) {
	c := cfg.(*Config)
	return &logsConnector{
		cfg:             c,
		logger:          set.Logger,
		defaultConsumer: logs,
		settings: exporter.Settings{
			ID:                set.ID,
			TelemetrySettings: set.TelemetrySettings,
			BuildInfo:         set.BuildInfo,
		},
	}, nil
}
