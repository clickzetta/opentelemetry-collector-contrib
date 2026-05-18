// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// TenantRecords holds recorded telemetry stats for a single tenant.
type TenantRecords struct {
	BatchCount  int
	RecordCount int
}

// MockRouter records routed telemetry in memory for testing.
type MockRouter struct {
	mu      sync.Mutex
	records map[string]*TenantRecords
	logger  *zap.Logger
}

// NewMockRouter creates a new MockRouter.
func NewMockRouter(logger *zap.Logger) *MockRouter {
	return &MockRouter{
		records: make(map[string]*TenantRecords),
		logger:  logger,
	}
}

// Route records the batch in memory and logs the routing event.
func (m *MockRouter) Route(_ context.Context, telType TelemetryType, tenant *TenantConfig, data any) error {
	count := countRecords(telType, data)

	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.records[tenant.TenantName]
	if !ok {
		rec = &TenantRecords{}
		m.records[tenant.TenantName] = rec
	}
	rec.BatchCount++
	rec.RecordCount += count

	m.logger.Info(fmt.Sprintf("[MOCK] Routed %d %s records to tenant %s", count, telType, tenant.TenantName))
	return nil
}

// GetRecords returns the recorded telemetry stats per tenant.
func (m *MockRouter) GetRecords() map[string]*TenantRecords {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to avoid races
	result := make(map[string]*TenantRecords, len(m.records))
	for k, v := range m.records {
		copied := *v
		result[k] = &copied
	}
	return result
}

// Close is a no-op for MockRouter.
func (m *MockRouter) Close() error {
	return nil
}

// countRecords returns the number of records in the telemetry data.
func countRecords(telType TelemetryType, data any) int {
	switch telType {
	case TelemetryTypeLogs:
		if ld, ok := data.(plog.Logs); ok {
			return ld.LogRecordCount()
		}
	case TelemetryTypeTraces:
		if td, ok := data.(ptrace.Traces); ok {
			return td.SpanCount()
		}
	case TelemetryTypeMetrics:
		if md, ok := data.(pmetric.Metrics); ok {
			return md.DataPointCount()
		}
	}
	return 0
}
