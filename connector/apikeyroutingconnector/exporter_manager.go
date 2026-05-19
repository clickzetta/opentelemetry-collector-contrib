// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// componentFactoryHost is an interface that the component.Host must implement
// to provide factory lookup capabilities for dynamic exporter creation.
type componentFactoryHost interface {
	component.Host
	GetFactory(kind component.Kind, componentType component.Type) component.Factory
}

// configHashInput is the struct serialized to JSON for hashing.
// Using a struct ensures deterministic field ordering.
type configHashInput struct {
	ExporterType   string         `json:"exporter_type"`
	ExporterConfig map[string]any `json:"exporter_config"`
}

// configHash computes a stable hash of (exporterType + exporterConfig) for exporter caching.
// The hash is deterministic: same inputs always produce the same hash.
// Go's json.Marshal sorts map keys alphabetically, ensuring determinism for map[string]any.
func configHash(exporterType string, exporterConfig map[string]any) string {
	input := configHashInput{
		ExporterType:   exporterType,
		ExporterConfig: exporterConfig,
	}
	// json.Marshal sorts map keys, so the output is deterministic.
	// Error is ignored because configHashInput contains only JSON-safe types.
	data, _ := json.Marshal(input)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// managedExporter wraps a dynamically created exporter with lifecycle metadata.
type managedExporter struct {
	logsExporter    exporter.Logs
	tracesExporter  exporter.Traces
	metricsExporter exporter.Metrics
	lastUsed        time.Time
	configHash      string
}

// ExporterManager creates, caches, and manages dynamically-created exporter instances.
type ExporterManager struct {
	mu          sync.RWMutex
	exporters   map[string]*managedExporter // keyed by hash(exporter_type + exporter_config)
	host        componentFactoryHost
	logger      *zap.Logger
	idleTimeout time.Duration
	settings    exporter.Settings
	nowFunc     func() time.Time // injected for testability
}

// NewExporterManager creates a manager with the given idle timeout.
func NewExporterManager(host componentFactoryHost, settings exporter.Settings, logger *zap.Logger, idleTimeout time.Duration) *ExporterManager {
	return &ExporterManager{
		exporters:   make(map[string]*managedExporter),
		host:        host,
		logger:      logger,
		idleTimeout: idleTimeout,
		settings:    settings,
		nowFunc:     time.Now,
	}
}

// GetOrCreateLogs returns a cached logs exporter or creates a new one.
func (m *ExporterManager) GetOrCreateLogs(ctx context.Context, exporterType string, exporterConfig map[string]any) (exporter.Logs, error) {
	hash := configHash(exporterType, exporterConfig)

	// Fast path: check if exporter already exists.
	m.mu.RLock()
	if me, ok := m.exporters[hash]; ok && me.logsExporter != nil {
		me.lastUsed = m.nowFunc()
		m.mu.RUnlock()
		return me.logsExporter, nil
	}
	m.mu.RUnlock()

	// Slow path: create a new exporter.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if me, ok := m.exporters[hash]; ok && me.logsExporter != nil {
		me.lastUsed = m.nowFunc()
		return me.logsExporter, nil
	}

	factory, cfg, err := m.buildExporterConfig(exporterType, exporterConfig)
	if err != nil {
		return nil, err
	}

	exp, err := factory.CreateLogs(ctx, m.settings, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs exporter %q: %w", exporterType, err)
	}

	if err := exp.Start(ctx, m.host); err != nil {
		return nil, fmt.Errorf("failed to start logs exporter %q: %w", exporterType, err)
	}

	me := m.getOrInitManagedExporter(hash)
	me.logsExporter = exp
	me.lastUsed = m.nowFunc()

	m.logger.Debug("created dynamic logs exporter", zap.String("type", exporterType), zap.String("hash", hash))
	return exp, nil
}

// GetOrCreateTraces returns a cached traces exporter or creates a new one.
func (m *ExporterManager) GetOrCreateTraces(ctx context.Context, exporterType string, exporterConfig map[string]any) (exporter.Traces, error) {
	hash := configHash(exporterType, exporterConfig)

	// Fast path: check if exporter already exists.
	m.mu.RLock()
	if me, ok := m.exporters[hash]; ok && me.tracesExporter != nil {
		me.lastUsed = m.nowFunc()
		m.mu.RUnlock()
		return me.tracesExporter, nil
	}
	m.mu.RUnlock()

	// Slow path: create a new exporter.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if me, ok := m.exporters[hash]; ok && me.tracesExporter != nil {
		me.lastUsed = m.nowFunc()
		return me.tracesExporter, nil
	}

	factory, cfg, err := m.buildExporterConfig(exporterType, exporterConfig)
	if err != nil {
		return nil, err
	}

	exp, err := factory.CreateTraces(ctx, m.settings, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create traces exporter %q: %w", exporterType, err)
	}

	if err := exp.Start(ctx, m.host); err != nil {
		return nil, fmt.Errorf("failed to start traces exporter %q: %w", exporterType, err)
	}

	me := m.getOrInitManagedExporter(hash)
	me.tracesExporter = exp
	me.lastUsed = m.nowFunc()

	m.logger.Debug("created dynamic traces exporter", zap.String("type", exporterType), zap.String("hash", hash))
	return exp, nil
}

// GetOrCreateMetrics returns a cached metrics exporter or creates a new one.
func (m *ExporterManager) GetOrCreateMetrics(ctx context.Context, exporterType string, exporterConfig map[string]any) (exporter.Metrics, error) {
	hash := configHash(exporterType, exporterConfig)

	// Fast path: check if exporter already exists.
	m.mu.RLock()
	if me, ok := m.exporters[hash]; ok && me.metricsExporter != nil {
		me.lastUsed = m.nowFunc()
		m.mu.RUnlock()
		return me.metricsExporter, nil
	}
	m.mu.RUnlock()

	// Slow path: create a new exporter.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock.
	if me, ok := m.exporters[hash]; ok && me.metricsExporter != nil {
		me.lastUsed = m.nowFunc()
		return me.metricsExporter, nil
	}

	factory, cfg, err := m.buildExporterConfig(exporterType, exporterConfig)
	if err != nil {
		return nil, err
	}

	exp, err := factory.CreateMetrics(ctx, m.settings, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics exporter %q: %w", exporterType, err)
	}

	if err := exp.Start(ctx, m.host); err != nil {
		return nil, fmt.Errorf("failed to start metrics exporter %q: %w", exporterType, err)
	}

	me := m.getOrInitManagedExporter(hash)
	me.metricsExporter = exp
	me.lastUsed = m.nowFunc()

	m.logger.Debug("created dynamic metrics exporter", zap.String("type", exporterType), zap.String("hash", hash))
	return exp, nil
}

// EvictIdle shuts down and removes exporters idle for longer than idleTimeout.
// Called periodically by a background goroutine.
func (m *ExporterManager) EvictIdle() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.nowFunc()
	for hash, me := range m.exporters {
		if now.Sub(me.lastUsed) > m.idleTimeout {
			m.logger.Info("evicting idle exporter", zap.String("hash", hash), zap.Duration("idle", now.Sub(me.lastUsed)))
			m.shutdownManagedExporter(me)
			delete(m.exporters, hash)
		}
	}
}

// Shutdown shuts down all managed exporters.
func (m *ExporterManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs error
	for hash, me := range m.exporters {
		m.logger.Debug("shutting down managed exporter", zap.String("hash", hash))
		if err := m.shutdownManagedExporterWithCtx(ctx, me); err != nil {
			errs = multierr.Append(errs, err)
		}
		delete(m.exporters, hash)
	}
	return errs
}

// buildExporterConfig looks up the factory and builds the config for the given exporter type.
func (m *ExporterManager) buildExporterConfig(exporterType string, exporterConfig map[string]any) (exporter.Factory, component.Config, error) {
	componentType := component.MustNewType(exporterType)
	f := m.host.GetFactory(component.KindExporter, componentType)
	if f == nil {
		return nil, nil, fmt.Errorf("exporter factory not found for type %q", exporterType)
	}

	factory, ok := f.(exporter.Factory)
	if !ok {
		return nil, nil, fmt.Errorf("factory for type %q is not an exporter factory", exporterType)
	}

	cfg := factory.CreateDefaultConfig()

	// Unmarshal the exporter_config map into the default config struct.
	cm := confmap.NewFromStringMap(exporterConfig)
	if err := cm.Unmarshal(cfg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal exporter config for type %q: %w", exporterType, err)
	}

	return factory, cfg, nil
}

// getOrInitManagedExporter returns the existing managedExporter for the hash or creates a new one.
// Must be called with m.mu held for writing.
func (m *ExporterManager) getOrInitManagedExporter(hash string) *managedExporter {
	me, ok := m.exporters[hash]
	if !ok {
		me = &managedExporter{configHash: hash}
		m.exporters[hash] = me
	}
	return me
}

// shutdownManagedExporter shuts down all exporter instances in a managedExporter using a background context.
func (m *ExporterManager) shutdownManagedExporter(me *managedExporter) {
	ctx := context.Background()
	if err := m.shutdownManagedExporterWithCtx(ctx, me); err != nil {
		m.logger.Error("error shutting down managed exporter", zap.String("hash", me.configHash), zap.Error(err))
	}
}

// shutdownManagedExporterWithCtx shuts down all exporter instances in a managedExporter.
func (m *ExporterManager) shutdownManagedExporterWithCtx(ctx context.Context, me *managedExporter) error {
	var errs error
	if me.logsExporter != nil {
		if err := me.logsExporter.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("logs exporter shutdown: %w", err))
		}
	}
	if me.tracesExporter != nil {
		if err := me.tracesExporter.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("traces exporter shutdown: %w", err))
		}
	}
	if me.metricsExporter != nil {
		if err := me.metricsExporter.Shutdown(ctx); err != nil {
			errs = multierr.Append(errs, fmt.Errorf("metrics exporter shutdown: %w", err))
		}
	}
	return errs
}
