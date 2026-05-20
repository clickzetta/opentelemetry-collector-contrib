// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// metricsConnector routes metrics based on API key resolution.
type metricsConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	dataCache       *DataCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Metrics
	settings        exporter.Settings
	stopEviction    chan struct{}
}

func (c *metricsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeMetrics implements consumer.Metrics.
// It resolves the route for the API key and either forwards metrics immediately
// or buffers them in the per-key DataCache for batch sending.
func (c *metricsConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// Resolve route using the shared resolution pipeline.
	entries := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)

	// If nil entries returned, forward to default consumer.
	if entries == nil {
		return c.forwardMetricsToDefault(ctx, md)
	}

	// If DataCache is enabled, buffer the data per key for batch sending.
	if c.dataCache != nil {
		key, _ := extractAPIKey(ctx, c.cfg.KeyHeader)
		return c.dataCache.AddMetrics(ctx, key, md)
	}

	// No buffering — forward immediately to each exporter.
	// NOTE: The same md is passed to each exporter. This is safe because
	// Capabilities() declares MutatesData: false, so exporters must not modify it.
	var lastErr error
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateMetrics(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			c.logger.Error("failed to get or create metrics exporter, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if err := c.forwardMetricsToDefault(ctx, md); err != nil {
				lastErr = err
			}
			continue
		}

		if consumeErr := exp.ConsumeMetrics(ctx, md); consumeErr != nil {
			c.logger.Debug("downstream exporter returned error",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(consumeErr),
			)
			lastErr = consumeErr
		}
	}

	return lastErr
}

// flushMetrics is the callback invoked by DataCache when a metrics buffer is flushed.
func (c *metricsConnector) flushMetrics(ctx context.Context, key string, md pmetric.Metrics) error {
	entries := resolveRouteForKey(ctx, key, c.cfg, c.cache, c.keyClient, c.logger)

	if entries == nil {
		return c.forwardMetricsToDefault(ctx, md)
	}

	var lastErr error
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateMetrics(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			c.logger.Error("failed to get or create metrics exporter during flush, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if err := c.forwardMetricsToDefault(ctx, md); err != nil {
				lastErr = err
			}
			continue
		}

		if consumeErr := exp.ConsumeMetrics(ctx, md); consumeErr != nil {
			c.logger.Debug("downstream exporter returned error during flush",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(consumeErr),
			)
			lastErr = consumeErr
		}
	}

	return lastErr
}

// forwardMetricsToDefault sends metrics to the default consumer.
func (c *metricsConnector) forwardMetricsToDefault(ctx context.Context, md pmetric.Metrics) error {
	if c.defaultConsumer != nil {
		return c.defaultConsumer.ConsumeMetrics(ctx, md)
	}
	c.logger.Warn("no default consumer configured, dropping metrics",
		zap.Int("data_points", md.DataPointCount()),
	)
	return nil
}

// Start initializes the metrics connector resources.
func (c *metricsConnector) Start(_ context.Context, host component.Host) error {
	c.keyClient = NewKeyServiceClient(c.cfg.KeyServiceURL, c.cfg.KeyServiceTimeout)
	c.cache = NewRouteCache(c.cfg.CacheTTL)

	factoryHost, ok := host.(componentFactoryHost)
	if !ok {
		c.logger.Warn("host does not support factory lookup, dynamic exporter creation will not be available")
	} else {
		c.exporterManager = NewExporterManager(factoryHost, c.settings, c.logger, c.cfg.ExporterIdleTimeout)
	}

	// Launch background eviction goroutine if ExporterManager is available.
	if c.exporterManager != nil {
		c.stopEviction = make(chan struct{})
		ticker := time.NewTicker(c.cfg.ExporterIdleTimeout / 2)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					c.exporterManager.EvictIdle()
				case <-c.stopEviction:
					return
				}
			}
		}()
	}

	// Initialize DataCache if batch_size > 0.
	if c.cfg.BatchSize > 0 {
		c.dataCache = NewDataCache(
			DataCacheConfig{
				BatchSize:     c.cfg.BatchSize,
				FlushInterval: c.cfg.FlushInterval,
			},
			c.logger,
			nil, // logs flush not used by metrics connector
			c.flushMetrics,
			nil, // traces flush not used by metrics connector
		)
		c.dataCache.Start()
	}

	return nil
}

// Shutdown releases the metrics connector resources.
func (c *metricsConnector) Shutdown(ctx context.Context) error {
	if c.stopEviction != nil {
		close(c.stopEviction)
	}

	var errs error
	if c.dataCache != nil {
		if err := c.dataCache.Stop(ctx); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	if c.exporterManager != nil {
		if err := c.exporterManager.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}
