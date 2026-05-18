// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

// gatewayMetricsExporter handles metrics in API key gateway mode.
type gatewayMetricsExporter struct {
	logger    *zap.Logger
	cfg       *Config
	cache     *KeyCache
	keyClient *KeyServiceClient
	router    Router
}

func newGatewayMetricsExporter(logger *zap.Logger, cfg *Config) *gatewayMetricsExporter {
	return &gatewayMetricsExporter{logger: logger, cfg: cfg}
}

func (e *gatewayMetricsExporter) start(_ context.Context, _ component.Host) error {
	e.keyClient = NewKeyServiceClient(e.cfg.APIKeyServiceURL)
	e.cache = NewKeyCache(e.cfg.CacheTTL)
	e.router = createRouter(e.cfg, e.logger)
	return nil
}

func (e *gatewayMetricsExporter) shutdown(_ context.Context) error {
	if e.router != nil {
		return e.router.Close()
	}
	return nil
}

func (e *gatewayMetricsExporter) pushMetricsData(ctx context.Context, md pmetric.Metrics) error {
	tenant, apiKey, err := resolveTenantConfig(ctx, e.cfg, e.cache, e.keyClient, e.logger)
	if err != nil {
		return err
	}

	injectTenantAttributesMetrics(md, tenant, apiKey)

	if err := e.router.Route(ctx, TelemetryTypeMetrics, tenant, md); err != nil {
		return err
	}

	e.logger.Debug("routed metrics batch",
		zap.String("tenant", tenant.TenantName),
		zap.Int("record_count", md.DataPointCount()))
	return nil
}

// injectTenantAttributesMetrics adds tenant identity resource attributes to all metric resources.
func injectTenantAttributesMetrics(md pmetric.Metrics, tenant *TenantConfig, apiKey string) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		attrs := md.ResourceMetrics().At(i).Resource().Attributes()
		attrs.PutStr("tenant.name", tenant.TenantName)
		attrs.PutStr("tenant.apikey_prefix", maskAPIKey(apiKey))
	}
}
