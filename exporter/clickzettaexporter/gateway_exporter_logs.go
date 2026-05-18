// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// gatewayLogsExporter handles logs in API key gateway mode.
type gatewayLogsExporter struct {
	logger    *zap.Logger
	cfg       *Config
	cache     *KeyCache
	keyClient *KeyServiceClient
	router    Router
}

func newGatewayLogsExporter(logger *zap.Logger, cfg *Config) *gatewayLogsExporter {
	return &gatewayLogsExporter{logger: logger, cfg: cfg}
}

func (e *gatewayLogsExporter) start(_ context.Context, _ component.Host) error {
	e.keyClient = NewKeyServiceClient(e.cfg.APIKeyServiceURL)
	e.cache = NewKeyCache(e.cfg.CacheTTL)
	e.router = createRouter(e.cfg, e.logger)
	return nil
}

func (e *gatewayLogsExporter) shutdown(_ context.Context) error {
	if e.router != nil {
		return e.router.Close()
	}
	return nil
}

func (e *gatewayLogsExporter) pushLogsData(ctx context.Context, ld plog.Logs) error {
	tenant, apiKey, err := resolveTenantConfig(ctx, e.cfg, e.cache, e.keyClient, e.logger)
	if err != nil {
		return err
	}

	injectTenantAttributesLogs(ld, tenant, apiKey)

	if err := e.router.Route(ctx, TelemetryTypeLogs, tenant, ld); err != nil {
		return err
	}

	e.logger.Debug("routed logs batch",
		zap.String("tenant", tenant.TenantName),
		zap.Int("record_count", ld.LogRecordCount()))
	return nil
}

// injectTenantAttributesLogs adds tenant identity resource attributes to all log resources.
func injectTenantAttributesLogs(ld plog.Logs, tenant *TenantConfig, apiKey string) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		attrs := ld.ResourceLogs().At(i).Resource().Attributes()
		attrs.PutStr("tenant.name", tenant.TenantName)
		attrs.PutStr("tenant.apikey_prefix", maskAPIKey(apiKey))
	}
}
