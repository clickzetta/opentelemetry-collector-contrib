// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

// tenantConn holds a per-tenant ClickZetta connection pair.
type tenantConn struct {
	db       *sql.DB
	conn     *goclickzetta.ClickzettaConn
	lastUsed time.Time
}

// RealRouter writes telemetry to actual ClickZetta backends.
type RealRouter struct {
	mu          sync.RWMutex
	connections map[string]*tenantConn
	cfg         *Config
	logger      *zap.Logger
	done        chan struct{}
}

// NewRealRouter creates a new RealRouter.
func NewRealRouter(cfg *Config, logger *zap.Logger) *RealRouter {
	r := &RealRouter{
		connections: make(map[string]*tenantConn),
		cfg:         cfg,
		logger:      logger,
		done:        make(chan struct{}),
	}
	go r.evictIdle()
	return r
}

// Route writes the telemetry batch to the tenant's ClickZetta backend.
// Lazily creates connections on first use. Reuses the shared write functions
// from exporter_write.go.
func (r *RealRouter) Route(ctx context.Context, telType TelemetryType, tenant *TenantConfig, data any) error {
	tc, err := r.getOrCreateConn(ctx, tenant)
	if err != nil {
		return fmt.Errorf("failed to get connection for tenant %s: %w", tenant.TenantName, err)
	}

	// Update lastUsed timestamp
	r.mu.Lock()
	tc.lastUsed = time.Now()
	r.mu.Unlock()

	switch telType {
	case TelemetryTypeLogs:
		ld, ok := data.(plog.Logs)
		if !ok {
			return fmt.Errorf("expected plog.Logs for logs telemetry type")
		}
		if err := writeLogs(tc.conn, r.cfg, ld); err != nil {
			return err
		}
		r.logger.Debug("Successfully wrote logs via RealRouter", zap.Int("log_records", ld.LogRecordCount()))
		return nil

	case TelemetryTypeTraces:
		td, ok := data.(ptrace.Traces)
		if !ok {
			return fmt.Errorf("expected ptrace.Traces for traces telemetry type")
		}
		if err := writeTraces(tc.conn, r.cfg, td); err != nil {
			return err
		}
		r.logger.Debug("Successfully wrote traces via RealRouter", zap.Int("spans", td.SpanCount()))
		return nil

	case TelemetryTypeMetrics:
		md, ok := data.(pmetric.Metrics)
		if !ok {
			return fmt.Errorf("expected pmetric.Metrics for metrics telemetry type")
		}
		if err := writeMetrics(tc.conn, r.cfg, md); err != nil {
			return err
		}
		r.logger.Debug("Successfully wrote metrics via RealRouter", zap.Int("data_points", md.DataPointCount()))
		return nil

	default:
		return fmt.Errorf("unknown telemetry type: %s", telType)
	}
}

// Close closes all active connections and stops the eviction goroutine.
func (r *RealRouter) Close() error {
	close(r.done)

	r.mu.Lock()
	defer r.mu.Unlock()

	for name, tc := range r.connections {
		if tc.db != nil {
			_ = tc.db.Close()
		}
		if tc.conn != nil {
			_ = tc.conn.Close()
		}
		r.logger.Debug("Closed connection for tenant", zap.String("tenant", name))
	}
	r.connections = make(map[string]*tenantConn)
	return nil
}

// evictIdle removes connections that have been idle for more than 30 minutes.
// It runs in a background goroutine and exits when the done channel is closed.
func (r *RealRouter) evictIdle() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.done:
			return
		case <-ticker.C:
			r.mu.Lock()
			for name, tc := range r.connections {
				if time.Since(tc.lastUsed) > 30*time.Minute {
					if tc.db != nil {
						_ = tc.db.Close()
					}
					if tc.conn != nil {
						_ = tc.conn.Close()
					}
					delete(r.connections, name)
					r.logger.Debug("Evicted idle connection for tenant", zap.String("tenant", name))
				}
			}
			r.mu.Unlock()
		}
	}
}

// getOrCreateConn returns an existing connection or lazily creates one.
func (r *RealRouter) getOrCreateConn(ctx context.Context, tenant *TenantConfig) (*tenantConn, error) {
	// Fast path: read lock check
	r.mu.RLock()
	if tc, ok := r.connections[tenant.TenantName]; ok {
		r.mu.RUnlock()
		return tc, nil
	}
	r.mu.RUnlock()

	// Slow path: write lock and create
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if tc, ok := r.connections[tenant.TenantName]; ok {
		return tc, nil
	}

	dsn := r.buildDSN(tenant)
	tc := &tenantConn{lastUsed: time.Now()}

	// Create tables if configured
	if r.cfg.CreateSchema {
		db, err := internal.NewSQLDB(dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open SQL connection for tenant %s: %w", tenant.TenantName, err)
		}
		tc.db = db
		if err := internal.CreateTables(ctx, db, r.cfg.LogsTableName, r.cfg.TracesTableName, r.cfg.MetricsTableName); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to create tables for tenant %s: %w", tenant.TenantName, err)
		}
		r.logger.Info("Created schema tables for tenant", zap.String("tenant", tenant.TenantName))
	}

	// Create bulkload connection
	conn, err := internal.NewBulkloadConn(dsn)
	if err != nil {
		if tc.db != nil {
			_ = tc.db.Close()
		}
		return nil, fmt.Errorf("failed to open bulkload connection for tenant %s: %w", tenant.TenantName, err)
	}
	tc.conn = conn

	r.connections[tenant.TenantName] = tc
	r.logger.Info("Created new connection for tenant", zap.String("tenant", tenant.TenantName))
	return tc, nil
}

// buildDSN constructs a DSN from the TenantConfig using the main config's protocol.
func (r *RealRouter) buildDSN(tenant *TenantConfig) string {
	return tenant.Username + ":" + tenant.Password + "@" + r.cfg.Protocol + "(" + tenant.Service + ")/" + tenant.Schema +
		"?virtualCluster=" + tenant.VirtualCluster + "&workspace=" + tenant.Workspace + "&instance=" + tenant.Instance
}
