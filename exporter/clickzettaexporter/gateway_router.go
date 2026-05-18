// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import "context"

// TelemetryType identifies the kind of telemetry being routed.
type TelemetryType string

const (
	TelemetryTypeLogs    TelemetryType = "logs"
	TelemetryTypeTraces  TelemetryType = "traces"
	TelemetryTypeMetrics TelemetryType = "metrics"
)

// Router defines the interface for routing telemetry to tenant backends.
type Router interface {
	// Route dispatches a telemetry batch to the appropriate backend.
	Route(ctx context.Context, telType TelemetryType, tenant *TenantConfig, data any) error
	// Close releases all resources held by the router.
	Close() error
}
