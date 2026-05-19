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
	"go.uber.org/zap"
)

// tracesConnector routes traces based on API key resolution.
type tracesConnector struct {
	logger          *zap.Logger
	cfg             *Config
	cache           *RouteCache
	keyClient       *KeyServiceClient
	exporterManager *ExporterManager
	defaultConsumer consumer.Traces
	host            component.Host
	settings        exporter.Settings
	stopEviction    chan struct{}
}

// Capabilities implements consumer.Traces.
func (c *tracesConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Start initializes the connector resources.
func (c *tracesConnector) Start(ctx context.Context, host component.Host) error {
	c.host = host

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

	// Shut down the ExporterManager (shuts down all dynamic exporters).
	if c.exporterManager != nil {
		if err := c.exporterManager.Shutdown(ctx); err != nil {
			c.logger.Error("error shutting down exporter manager", zap.Error(err))
		}
	}

	// Close the KeyServiceClient (the HTTP client doesn't need explicit close,
	// but we nil the reference for safety).
	c.keyClient = nil

	return nil
}

// ConsumeTraces routes traces based on API key resolution.
func (c *tracesConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// Step 1: Resolve route using the key resolution pipeline.
	entries, err := resolveRoute(ctx, c.cfg, c.cache, c.keyClient, c.logger)
	if err != nil {
		c.logger.Error("unexpected error during route resolution, routing to default",
			zap.Error(err),
		)
		return c.forwardToDefault(ctx, td)
	}

	// Step 2: If nil entries returned, forward to default consumer.
	if entries == nil {
		return c.forwardToDefault(ctx, td)
	}

	// Step 3: For each RouteEntry, get or create exporter and forward traces.
	var lastErr error
	for _, entry := range entries {
		if c.exporterManager == nil {
			c.logger.Warn("exporter manager not available, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
			)
			if err := c.forwardToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		exp, err := c.exporterManager.GetOrCreateTraces(ctx, entry.ExporterType, entry.ExporterConfig)
		if err != nil {
			// Exporter creation failed — fall back to default consumer, log error.
			c.logger.Error("failed to get or create traces exporter, routing to default",
				zap.String("pipeline_id", entry.PipelineID),
				zap.String("exporter_type", entry.ExporterType),
				zap.Error(err),
			)
			if err := c.forwardToDefault(ctx, td); err != nil {
				lastErr = err
			}
			continue
		}

		// Step 4: Forward traces to the exporter.
		if err := exp.ConsumeTraces(ctx, td); err != nil {
			// Step 6: Propagate downstream errors.
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

// forwardToDefault sends traces to the default consumer.
func (c *tracesConnector) forwardToDefault(ctx context.Context, td ptrace.Traces) error {
	if c.defaultConsumer == nil {
		c.logger.Warn("no default consumer configured, dropping traces",
			zap.Int("span_count", td.SpanCount()),
		)
		return nil
	}
	return c.defaultConsumer.ConsumeTraces(ctx, td)
}

// healthCheck performs a non-blocking connectivity check to the Key Service.
// Logs a warning if unreachable but does not fail startup.
func (c *tracesConnector) healthCheck(ctx context.Context) {
	// Use a simple resolve call with a dummy key to check connectivity.
	// We don't care about the result, only whether the service is reachable.
	_, err := c.keyClient.Resolve(ctx, "__health_check__")
	if err != nil {
		// Only network/timeout errors indicate unreachability.
		// 401/404 means the service is reachable (just the key is invalid).
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
