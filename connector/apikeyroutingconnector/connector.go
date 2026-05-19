// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"errors"

	"go.uber.org/zap"
)

// resolveRoute implements the key resolution pipeline:
// extract key → check cache → call Key Service → stale fallback → default.
//
// Returns the target RouteEntry slice or nil (route to default).
// A nil return with no error means the caller should route to the default pipeline.
func resolveRoute(
	ctx context.Context,
	cfg *Config,
	cache *RouteCache,
	keyClient *KeyServiceClient,
	logger *zap.Logger,
) ([]*RouteEntry, error) {
	// Step 1: Extract API key from context.
	key, err := extractAPIKey(ctx, cfg.KeyHeader)
	if err != nil {
		if errors.Is(err, errMissingKey) {
			logger.Warn("API key header missing from request metadata, routing to default",
				zap.String("header", cfg.KeyHeader),
			)
			return nil, nil
		}
		if errors.Is(err, errEmptyKey) {
			logger.Warn("API key header value is empty after trimming, routing to default",
				zap.String("header", cfg.KeyHeader),
			)
			return nil, nil
		}
		// Unexpected extraction error — route to default.
		logger.Warn("unexpected error extracting API key, routing to default",
			zap.String("header", cfg.KeyHeader),
			zap.Error(err),
		)
		return nil, nil
	}

	maskedKey := maskAPIKey(key)

	// Step 2: Check cache for a fresh entry.
	entry, fresh := cache.Get(key)
	if entry != nil && fresh {
		logger.Debug("cache hit (fresh), routing to cached pipeline",
			zap.String("key", maskedKey),
			zap.String("pipeline_id", entry.PipelineID),
		)
		return []*RouteEntry{entry}, nil
	}

	// Step 3: Call Key Service to resolve the key.
	resp, err := keyClient.Resolve(ctx, key)
	if err == nil {
		// Success (200 OK): parse response and update cache.
		entries, parseErr := parseRouteEntry(resp, cfg.DefaultExporterType)
		if parseErr != nil {
			// Parse error after successful HTTP response — treat as retryable.
			logger.Warn("failed to parse Key Service response, attempting stale fallback",
				zap.String("key", maskedKey),
				zap.Error(parseErr),
			)
			return staleFallbackOrDefault(entry, maskedKey, logger), nil
		}

		// Store each entry in cache.
		for _, e := range entries {
			cache.Set(key, e)
		}

		logger.Debug("Key Service resolved successfully, cache updated",
			zap.String("key", maskedKey),
			zap.String("pipeline_id", entries[0].PipelineID),
			zap.Int("pipeline_count", len(entries)),
		)
		return entries, nil
	}

	// Step 4: Handle Key Service errors.
	if errors.Is(err, errPermanentKeyInvalid) {
		// Permanent error (401/404): key is invalid, route to default.
		logger.Warn("API key is invalid (permanent error from Key Service), routing to default",
			zap.String("key", maskedKey),
			zap.Error(err),
		)
		return nil, nil
	}

	// Retryable error (5xx/timeout/network): attempt stale cache fallback.
	if isRetryable(err) {
		logger.Warn("Key Service unavailable (retryable error), attempting stale fallback",
			zap.String("key", maskedKey),
			zap.Error(err),
		)
		return staleFallbackOrDefault(entry, maskedKey, logger), nil
	}

	// Unknown error type — treat as retryable and attempt stale fallback.
	logger.Warn("unexpected Key Service error, attempting stale fallback",
		zap.String("key", maskedKey),
		zap.Error(err),
	)
	return staleFallbackOrDefault(entry, maskedKey, logger), nil
}

// staleFallbackOrDefault returns the stale cache entry if available,
// or nil (route to default) if no stale entry exists.
func staleFallbackOrDefault(staleEntry *RouteEntry, maskedKey string, logger *zap.Logger) []*RouteEntry {
	if staleEntry != nil {
		logger.Warn("serving stale cache entry due to Key Service unavailability",
			zap.String("key", maskedKey),
			zap.String("pipeline_id", staleEntry.PipelineID),
		)
		return []*RouteEntry{staleEntry}
	}

	logger.Warn("no stale cache entry available, routing to default",
		zap.String("key", maskedKey),
	)
	return nil
}
