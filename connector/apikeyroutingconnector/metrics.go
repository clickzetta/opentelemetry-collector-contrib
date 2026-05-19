// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// metricsConnector routes metrics based on API key resolution.
type metricsConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Metrics
}

func (c *metricsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics implements consumer.Metrics.
// It resolves the route for the API key and forwards metrics to the appropriate exporter.
func (c *metricsConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Step 1: Resolve route using the shared resolution pipeline.
	entries, err := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)
	if err != nil {
		return err
	}

	// Step 2: If nil entries returned, forward to default consumer.
	if entries == nil {
		if c.defaultConsumer != nil {
			return c.defaultConsumer.ConsumeMetrics(ctx, md)
		}
		c.logger.Warn("no default consumer configured, dropping metrics")
		return nil
	}

	// Step 3: For each RouteEntry, get or create the metrics exporter and forward.
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateMetrics(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			// Step 5: If exporter creation fails, fall back to default consumer and log error.
			c.logger.Error("failed to get or create metrics exporter, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if c.defaultConsumer != nil {
				return c.defaultConsumer.ConsumeMetrics(ctx, md)
			}
			return expErr
		}

		// Step 4: Forward metrics to the exporter.
		if consumeErr := exp.ConsumeMetrics(ctx, md); consumeErr != nil {
			// Step 6: Propagate any downstream errors.
			return consumeErr
		}
	}

	return nil
}

// Start initializes the metrics connector resources.
func (c *metricsConnector) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown releases the metrics connector resources.
func (c *metricsConnector) Shutdown(ctx context.Context) error {
	if c.exporterManager != nil {
		return c.exporterManager.Shutdown(ctx)
	}
	return nil
}
