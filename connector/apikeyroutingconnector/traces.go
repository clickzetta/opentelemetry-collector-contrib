// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// tracesConnector routes traces based on API key resolution.
type tracesConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	dataCache       *DataCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Traces
	settings        exporter.Settings
	stopEviction    chan struct{}
}

// Capabilities implements consumer.Traces.
func (c *tracesConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Start initializes the connector resources.
func (c *tracesConnector) Start(ctx context.Context, host component.Host) error {
	// Initialize the KeyServiceClient.
	c.keyClient = NewKeyServiceClient(c.cfg.KeyServiceURL, c.cfg.KeyServiceTimeout)

	// Initialize the RouteCache.
	c.cache = NewRouteCache(c.cfg.CacheTTL)

	// Initialize the ExporterManager (requires component.Host for factory lookup).
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
			nil, // logs flush not used by traces connector
			nil, // metrics flush not used by traces connector
			c.flushTraces,
		)
		c.dataCache.Start()
	}

	// Perform a non-blocking health check to Key Service.
	go c.healthCheck(ctx)

	return nil
}

// Shutdown releases all connector resources.
func (c *tracesConnector) Shutdown(ctx context.Context) error {
	// Stop the background eviction goroutine.
	if c.stopEviction != nil {
		close(c.stopEviction)
	}

	var errs error

	// Stop the DataCache (flushes remaining data).
	if c.dataCache != nil {
		if err := c.dataCache.Stop(ctx); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	// Shut down the ExporterManager (shuts down all dynamic exporters).
	if c.exporterManager != nil {
		if err := c.exporterManager.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, err)
		}
	}

	c.keyClient = nil
	return errs
}

// ConsumeTraces routes traces based on API key resolution.
func (c *tracesConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Resolve route using the key resolution pipeline.
	entries := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)

	// If nil entries returned, forward to default consumer.
	if entries == nil {
		return c.forwardTracesToDefault(ctx, td)
	}

	// If DataCache is enabled, buffer the data per key for batch sending.
	if c.dataCache != nil {
		key, _ := extractAPIKey(ctx, c.cfg.KeyHeader)
		return c.dataCache.AddTraces(ctx, key, td)
	}

	// No buffering — forward immediately to each exporter.
	// NOTE: The same td is passed to each exporter. This is safe because
	// Capabilities() declares MutatesData: false, so exporters must not modify it.
	var lastErr error
	for _, entry := range entries {
		if c.exporterManager == nil {
			c.logger.Warn("exporter manager not available, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
			)
			if err := c.forwardTracesToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		exp, err := c.exporterManager.GetOrCreateTraces(ctx, entry.ExporterType, entry.ExporterConfig)
		if err != nil {
			c.logger.Error("failed to get or create traces exporter, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
				zap.String("exporter_type", entry.ExporterType),
				zap.Error(err),
			)
			if err := c.forwardTracesToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		if err := exp.ConsumeTraces(ctx, td); err != nil {
			c.logger.Debug("downstream exporter returned error",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(err),
			)
			lastErr = err
		} else {
			c.logger.Debug("traces routed successfully",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Int("span_count", td.SpanCount()),
			)
		}
	}

	return lastErr
}

// flushTraces is the callback invoked by DataCache when a traces buffer is flushed.
func (c *tracesConnector) flushTraces(ctx context.Context, key string, td ptrace.Traces) error {
	entries := resolveRouteForKey(ctx, key, c.cfg, c.cache, c.keyClient, c.logger)

	if entries == nil {
		return c.forwardTracesToDefault(ctx, td)
	}

	var lastErr error
	for _, entry := range entries {
		if c.exporterManager == nil {
			c.logger.Warn("exporter manager not available, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
			)
			if err := c.forwardTracesToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		exp, err := c.exporterManager.GetOrCreateTraces(ctx, entry.ExporterType, entry.ExporterConfig)
		if err != nil {
			c.logger.Error("failed to get or create traces exporter during flush, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
				zap.String("exporter_type", entry.ExporterType),
				zap.Error(err),
			)
			if err := c.forwardTracesToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		if err := exp.ConsumeTraces(ctx, td); err != nil {
			c.logger.Debug("downstream exporter returned error during flush",
				zap.String("pipeline_id", entry.PipelineID),
				zap.Error(err),
			)
			lastErr = err
		}
	}

	return lastErr
}

// forwardTracesToDefault sends traces to the default consumer.
func (c *tracesConnector) forwardTracesToDefault(ctx context.Context, td ptrace.Traces) error {
	if c.defaultConsumer == nil {
		c.logger.Warn("no default consumer configured, dropping traces",
			zap.Int("span_count", td.SpanCount()),
		)
		return nil
	}
	return c.defaultConsumer.ConsumeTraces(ctx, td)
}

// healthCheck performs a non-blocking connectivity check to the Key Service.
func (c *tracesConnector) healthCheck(ctx context.Context) {
	_, err := c.keyClient.Resolve(ctx, "__health_check__")
	if err != nil {
		if isRetryable(err) {
			c.logger.Warn("Key Service health check failed, service may be unreachable",
				zap.String("url", c.cfg.KeyServiceURL),
				zap.Error(err),
			)
		} else {
			c.logger.Debug("Key Service health check completed (service reachable)",
				zap.String("url", c.cfg.KeyServiceURL),
			)
		}
	} else {
		c.logger.Debug("Key Service health check completed successfully",
			zap.String("url", c.cfg.KeyServiceURL),
		)
	}
}
