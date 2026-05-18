// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

// gatewayTracesExporter handles traces in API key gateway mode.
type gatewayTracesExporter struct {
	logger    *zap.Logger
	cfg       *Config
	cache     *KeyCache
	keyClient *KeyServiceClient
	router    Router
}

func newGatewayTracesExporter(logger *zap.Logger, cfg *Config) *gatewayTracesExporter {
	return &gatewayTracesExporter{logger: logger, cfg: cfg}
}

func (e *gatewayTracesExporter) start(_ context.Context, _ component.Host) error {
	e.keyClient = NewKeyServiceClient(e.cfg.APIKeyServiceURL)
	e.cache = NewKeyCache(e.cfg.CacheTTL)
	e.router = createRouter(e.cfg, e.logger)
	return nil
}

func (e *gatewayTracesExporter) shutdown(_ context.Context) error {
	if e.router != nil {
		return e.router.Close()
	}
	return nil
}

func (e *gatewayTracesExporter) pushTraceData(ctx context.Context, td ptrace.Traces) error {
	tenant, apiKey, err := resolveTenantConfig(ctx, e.cfg, e.cache, e.keyClient, e.logger)
	if err != nil {
		return err
	}

	injectTenantAttributesTraces(td, tenant, apiKey)

	if err := e.router.Route(ctx, TelemetryTypeTraces, tenant, td); err != nil {
		return err
	}

	e.logger.Debug("routed traces batch",
		zap.String("tenant", tenant.TenantName),
		zap.Int("record_count", td.SpanCount()))
	return nil
}

// injectTenantAttributesTraces adds tenant identity resource attributes to all trace resources.
func injectTenantAttributesTraces(td ptrace.Traces, tenant *TenantConfig, apiKey string) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		attrs := td.ResourceSpans().At(i).Resource().Attributes()
		attrs.PutStr("tenant.name", tenant.TenantName)
		attrs.PutStr("tenant.apikey_prefix", maskAPIKey(apiKey))
	}
}
