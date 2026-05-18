// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.uber.org/zap"
)

// createRouter creates the appropriate Router based on config.
func createRouter(cfg *Config, logger *zap.Logger) Router {
	if cfg.RouterMode == "mock" {
		return NewMockRouter(logger)
	}
	return NewRealRouter(cfg, logger)
}

// resolveTenantConfig implements the key resolution pipeline:
// extract API key → check cache → call Key Service → fall back to stale cache.
func resolveTenantConfig(ctx context.Context, cfg *Config, cache *KeyCache, keyClient *KeyServiceClient, logger *zap.Logger) (*TenantConfig, string, error) {
	apiKey, err := extractAPIKey(ctx, cfg.APIKeyHeader)
	if err != nil {
		return nil, "", err
	}

	// Check cache first.
	if config, fresh := cache.Get(apiKey); fresh {
		return config, apiKey, nil
	}

	// Cache miss or stale — call Key Service.
	config, err := keyClient.Resolve(ctx, apiKey)
	if err == nil {
		// Success: store in cache and return.
		cache.Set(apiKey, config)
		logger.Debug("refreshed tenant config from key service",
			zap.String("tenant", config.TenantName))
		return config, apiKey, nil
	}

	// Handle errors from Key Service.
	if consumererror.IsPermanent(err) || errors.Is(err, errInvalidKey) {
		// Invalid key — permanent error.
		logger.Warn("invalid API key",
			zap.String("key_prefix", maskAPIKey(apiKey)),
			zap.Error(err))
		return nil, apiKey, consumererror.NewPermanent(err)
	}

	// Retryable error (service unavailable) — try stale cache.
	if staleConfig, _ := cache.Get(apiKey); staleConfig != nil {
		logger.Warn("key service unavailable, serving stale cache entry",
			zap.String("key_prefix", maskAPIKey(apiKey)),
			zap.Error(err))
		return staleConfig, apiKey, nil
	}

	// No stale cache entry — return retryable error.
	logger.Warn("key service unavailable and no cache entry",
		zap.String("key_prefix", maskAPIKey(apiKey)),
		zap.Error(err))
	return nil, apiKey, err
}
