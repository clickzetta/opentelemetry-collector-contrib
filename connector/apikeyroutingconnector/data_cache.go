// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// logsFlushFunc is called when the logs buffer for a key is flushed.
type logsFlushFunc func(ctx context.Context, key string, ld plog.Logs) error

// metricsFlushFunc is called when the metrics buffer for a key is flushed.
type metricsFlushFunc func(ctx context.Context, key string, md pmetric.Metrics) error

// tracesFlushFunc is called when the traces buffer for a key is flushed.
type tracesFlushFunc func(ctx context.Context, key string, td ptrace.Traces) error

// logsBuf holds buffered logs for a single API key.
type logsBuf struct {
	data  plog.Logs
	count int // number of log records
}

// metricsBuf holds buffered metrics for a single API key.
type metricsBuf struct {
	data  pmetric.Metrics
	count int // number of data points
}

// tracesBuf holds buffered traces for a single API key.
type tracesBuf struct {
	data  ptrace.Traces
	count int // number of spans
}

// DataCache buffers telemetry data per API key and flushes in batches.
// It provides per-key batching so that data from the same tenant is accumulated
// and sent together, reducing the number of downstream exporter calls.
type DataCache struct {
	mu sync.Mutex

	// Logs buffers keyed by API key.
	logsBufs map[string]*logsBuf
	// Metrics buffers keyed by API key.
	metricsBufs map[string]*metricsBuf
	// Traces buffers keyed by API key.
	tracesBufs map[string]*tracesBuf

	// Flush callbacks.
	logsFlush    logsFlushFunc
	metricsFlush metricsFlushFunc
	tracesFlush  tracesFlushFunc

	// Configuration.
	batchSize     int
	flushInterval time.Duration
	logger        *zap.Logger

	// Lifecycle.
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// DataCacheConfig holds configuration for the DataCache.
type DataCacheConfig struct {
	// BatchSize is the number of records/data points/spans to accumulate
	// per key before triggering a flush.
	// Must be > 0 when DataCache is used (guarded by the caller).
	BatchSize int

	// FlushInterval is the maximum time to wait before flushing a non-empty
	// buffer, even if BatchSize has not been reached.
	// Must be > 0 when DataCache is used (validated in Config.Validate).
	FlushInterval time.Duration
}

// NewDataCache creates a new DataCache with the given configuration and flush callbacks.
func NewDataCache(
	cfg DataCacheConfig,
	logger *zap.Logger,
	logsFlush logsFlushFunc,
	metricsFlush metricsFlushFunc,
	tracesFlush tracesFlushFunc,
) *DataCache {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}

	return &DataCache{
		logsBufs:     make(map[string]*logsBuf),
		metricsBufs:  make(map[string]*metricsBuf),
		tracesBufs:   make(map[string]*tracesBuf),
		logsFlush:    logsFlush,
		metricsFlush: metricsFlush,
		tracesFlush:  tracesFlush,
		batchSize:    cfg.BatchSize,
		flushInterval: cfg.FlushInterval,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

// Start launches the background flush goroutine.
func (dc *DataCache) Start() {
	dc.wg.Add(1)
	go dc.flushLoop()
}

// Stop signals the flush loop to stop and flushes all remaining data.
func (dc *DataCache) Stop(ctx context.Context) error {
	close(dc.stopCh)
	dc.wg.Wait()

	// Final flush of all remaining buffers.
	return dc.flushAll(ctx)
}

// AddLogs adds logs to the buffer for the given API key.
// If the buffer exceeds BatchSize, it is flushed immediately.
func (dc *DataCache) AddLogs(ctx context.Context, key string, ld plog.Logs) error {
	dc.mu.Lock()

	buf, exists := dc.logsBufs[key]
	if !exists {
		buf = &logsBuf{data: plog.NewLogs()}
		dc.logsBufs[key] = buf
	}

	// Append all ResourceLogs from incoming data to the buffer.
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		ld.ResourceLogs().At(i).CopyTo(buf.data.ResourceLogs().AppendEmpty())
	}
	buf.count += ld.LogRecordCount()

	// Check if we should flush.
	if buf.count >= dc.batchSize {
		flushData := buf.data
		flushKey := key
		// Reset buffer.
		dc.logsBufs[key] = &logsBuf{data: plog.NewLogs()}
		dc.mu.Unlock()

		return dc.logsFlush(ctx, flushKey, flushData)
	}

	dc.mu.Unlock()
	return nil
}

// AddMetrics adds metrics to the buffer for the given API key.
// If the buffer exceeds BatchSize, it is flushed immediately.
func (dc *DataCache) AddMetrics(ctx context.Context, key string, md pmetric.Metrics) error {
	dc.mu.Lock()

	buf, exists := dc.metricsBufs[key]
	if !exists {
		buf = &metricsBuf{data: pmetric.NewMetrics()}
		dc.metricsBufs[key] = buf
	}

	// Append all ResourceMetrics from incoming data to the buffer.
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		md.ResourceMetrics().At(i).CopyTo(buf.data.ResourceMetrics().AppendEmpty())
	}
	buf.count += md.DataPointCount()

	// Check if we should flush.
	if buf.count >= dc.batchSize {
		flushData := buf.data
		flushKey := key
		dc.metricsBufs[key] = &metricsBuf{data: pmetric.NewMetrics()}
		dc.mu.Unlock()

		return dc.metricsFlush(ctx, flushKey, flushData)
	}

	dc.mu.Unlock()
	return nil
}

// AddTraces adds traces to the buffer for the given API key.
// If the buffer exceeds BatchSize, it is flushed immediately.
func (dc *DataCache) AddTraces(ctx context.Context, key string, td ptrace.Traces) error {
	dc.mu.Lock()

	buf, exists := dc.tracesBufs[key]
	if !exists {
		buf = &tracesBuf{data: ptrace.NewTraces()}
		dc.tracesBufs[key] = buf
	}

	// Append all ResourceSpans from incoming data to the buffer.
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		td.ResourceSpans().At(i).CopyTo(buf.data.ResourceSpans().AppendEmpty())
	}
	buf.count += td.SpanCount()

	// Check if we should flush.
	if buf.count >= dc.batchSize {
		flushData := buf.data
		flushKey := key
		dc.tracesBufs[key] = &tracesBuf{data: ptrace.NewTraces()}
		dc.mu.Unlock()

		return dc.tracesFlush(ctx, flushKey, flushData)
	}

	dc.mu.Unlock()
	return nil
}

// flushLoop periodically flushes all non-empty buffers.
func (dc *DataCache) flushLoop() {
	defer dc.wg.Done()

	ticker := time.NewTicker(dc.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := dc.flushAll(context.Background()); err != nil {
				dc.logger.Error("error during periodic flush", zap.Error(err))
			}
		case <-dc.stopCh:
			return
		}
	}
}

// flushAll flushes all non-empty buffers for all keys and all signal types.
func (dc *DataCache) flushAll(ctx context.Context) error {
	dc.mu.Lock()

	// Collect all non-empty logs buffers.
	logsToFlush := make(map[string]plog.Logs)
	for key, buf := range dc.logsBufs {
		if buf.count > 0 {
			logsToFlush[key] = buf.data
			dc.logsBufs[key] = &logsBuf{data: plog.NewLogs()}
		}
	}

	// Collect all non-empty metrics buffers.
	metricsToFlush := make(map[string]pmetric.Metrics)
	for key, buf := range dc.metricsBufs {
		if buf.count > 0 {
			metricsToFlush[key] = buf.data
			dc.metricsBufs[key] = &metricsBuf{data: pmetric.NewMetrics()}
		}
	}

	// Collect all non-empty traces buffers.
	tracesToFlush := make(map[string]ptrace.Traces)
	for key, buf := range dc.tracesBufs {
		if buf.count > 0 {
			tracesToFlush[key] = buf.data
			dc.tracesBufs[key] = &tracesBuf{data: ptrace.NewTraces()}
		}
	}

	dc.mu.Unlock()

	// Flush outside the lock, collecting all errors.
	var errs error

	for key, data := range logsToFlush {
		dc.logger.Debug("flushing logs buffer",
			zap.String("key", maskAPIKey(key)),
			zap.Int("log_records", data.LogRecordCount()),
		)
		if err := dc.logsFlush(ctx, key, data); err != nil {
			dc.logger.Error("error flushing logs buffer",
				zap.String("key", maskAPIKey(key)),
				zap.Error(err),
			)
			errs = multierr.Append(errs, err)
		}
	}

	for key, data := range metricsToFlush {
		dc.logger.Debug("flushing metrics buffer",
			zap.String("key", maskAPIKey(key)),
			zap.Int("data_points", data.DataPointCount()),
		)
		if err := dc.metricsFlush(ctx, key, data); err != nil {
			dc.logger.Error("error flushing metrics buffer",
				zap.String("key", maskAPIKey(key)),
				zap.Error(err),
			)
			errs = multierr.Append(errs, err)
		}
	}

	for key, data := range tracesToFlush {
		dc.logger.Debug("flushing traces buffer",
			zap.String("key", maskAPIKey(key)),
			zap.Int("spans", data.SpanCount()),
		)
		if err := dc.tracesFlush(ctx, key, data); err != nil {
			dc.logger.Error("error flushing traces buffer",
				zap.String("key", maskAPIKey(key)),
				zap.Error(err),
			)
			errs = multierr.Append(errs, err)
		}
	}

	return errs
}
