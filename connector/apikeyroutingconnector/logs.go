// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// logsConnector routes logs based on API key resolution.
type logsConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Logs
}

func (c *logsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeLogs implements consumer.Logs.
// It resolves the route for the API key and forwards logs to the appropriate exporter.
func (c *logsConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Step 1: Resolve route using the shared resolution pipeline.
	entries, err := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)
	if err != nil {
		return err
	}

	// Step 2: If nil entries returned, forward to default consumer.
	if entries == nil {
		if c.defaultConsumer != nil {
			return c.defaultConsumer.ConsumeLogs(ctx, ld)
		}
		c.logger.Warn("no default consumer configured, dropping logs")
		return nil
	}

	// Step 3: For each RouteEntry, get or create the logs exporter and forward.
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateLogs(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			// Step 5: If exporter creation fails, fall back to default consumer and log error.
			c.logger.Error("failed to get or create logs exporter, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if c.defaultConsumer != nil {
				return c.defaultConsumer.ConsumeLogs(ctx, ld)
			}
			return expErr
		}

		// Step 4: Forward logs to the exporter.
		if consumeErr := exp.ConsumeLogs(ctx, ld); consumeErr != nil {
			// Step 6: Propagate any downstream errors.
			return consumeErr
		}
	}

	return nil
}

// Start initializes the logs connector resources.
func (c *logsConnector) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown releases the logs connector resources.
func (c *logsConnector) Shutdown(ctx context.Context) error {
	if c.exporterManager != nil {
		return c.exporterManager.Shutdown(ctx)
	}
	return nil
}
