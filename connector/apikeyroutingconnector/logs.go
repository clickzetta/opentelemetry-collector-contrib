// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// logsConnector routes logs based on API key resolution.
type logsConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	dataCache       *DataCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Logs
	settings        exporter.Settings
	stopEviction    chan struct{}
}

func (c *logsConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// ConsumeLogs implements consumer.Logs.
// It resolves the route for the API key and either forwards logs immediately
// or buffers them in the per-key DataCache for batch sending.
func (c *logsConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// Resolve route using the shared resolution pipeline.
	entries := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)

	// If nil entries returned, forward to default consumer.
	if entries == nil {
		return c.forwardLogsToDefault(ctx, ld)
	}

	// If DataCache is enabled, buffer the data per key for batch sending.
	if c.dataCache != nil {
		key, _ := extractAPIKey(ctx, c.cfg.KeyHeader)
		return c.dataCache.AddLogs(ctx, key, ld)
	}

	// No buffering — forward immediately to each exporter.
	// NOTE: The same ld is passed to each exporter. This is safe because
	// Capabilities() declares MutatesData: false, so exporters must not modify it.
	var lastErr error
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateLogs(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			c.logger.Error("failed to get or create logs exporter, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if err := c.forwardLogsToDefault(ctx, ld); err != nil {
				lastErr = err
			}
			continue
		}

		if consumeErr := exp.ConsumeLogs(ctx, ld); consumeErr != nil {
			c.logger.Debug("downstream exporter returned error",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(consumeErr),
			)
			lastErr = consumeErr
		}
	}

	return lastErr
}

// flushLogs is the callback invoked by DataCache when a logs buffer is flushed.
// It resolves the route for the key and sends the batched logs to the exporter.
func (c *logsConnector) flushLogs(ctx context.Context, key string, ld plog.Logs) error {
	entries := resolveRouteForKey(ctx, key, c.cfg, c.cache, c.keyClient, c.logger)

	if entries == nil {
		return c.forwardLogsToDefault(ctx, ld)
	}

	var lastErr error
	for _, entry := range entries {
		exp, expErr := c.exporterManager.GetOrCreateLogs(ctx, entry.ExporterType, entry.ExporterConfig)
		if expErr != nil {
			c.logger.Error("failed to get or create logs exporter during flush, falling back to default",
				zap.String("exporter_type", entry.ExporterType),
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(expErr),
			)
			if err := c.forwardLogsToDefault(ctx, ld); err != nil {
				lastErr = err
			}
			continue
		}

		if consumeErr := exp.ConsumeLogs(ctx, ld); consumeErr != nil {
			c.logger.Debug("downstream exporter returned error during flush",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(consumeErr),
			)
			lastErr = consumeErr
		}
	}

	return lastErr
}

// forwardLogsToDefault sends logs to the default consumer.
func (c *logsConnector) forwardLogsToDefault(ctx context.Context, ld plog.Logs) error {
	if c.defaultConsumer != nil {
		return c.defaultConsumer.ConsumeLogs(ctx, ld)
	}
	c.logger.Warn("no default consumer configured, dropping logs",
		zap.Int("log_records", ld.LogRecordCount()),
	)
	return nil
}

// Start initializes the logs connector resources.
func (c *logsConnector) Start(_ context.Context, host component.Host) error {
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
			c.flushLogs,
			nil, // metrics flush not used by logs connector
			nil, // traces flush not used by logs connector
		)
		c.dataCache.Start()
	}

	return nil
}

// Shutdown releases the logs connector resources.
func (c *logsConnector) Shutdown(ctx context.Context) error {
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
