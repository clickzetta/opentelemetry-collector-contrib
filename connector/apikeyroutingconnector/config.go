// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"errors"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pipeline"
)

var (
	errEmptyKeyServiceURL          = errors.New("key_service_url must not be empty")
	errMissingAPIKeyPlaceholder    = errors.New("key_service_url must contain {api_key} placeholder")
	errNonPositiveCacheTTL         = errors.New("cache_ttl must be a positive duration")
	errNonPositiveServiceTimeout   = errors.New("key_service_timeout must be a positive duration")
	errNonPositiveFlushInterval    = errors.New("flush_interval must be a positive duration when batch_size > 0")
)

// Config defines configuration for the API Key Routing Connector.
type Config struct {
	// KeyServiceURL is the full URL template for the Key Service API.
	// Must contain `{api_key}` placeholder which will be replaced with the actual key.
	// Example: "http://key-service:8080/v1/apikeys/{api_key}/exporter"
	// Required.
	KeyServiceURL string `mapstructure:"key_service_url"`

	// KeyHeader is the Client_Metadata key to extract the API key from.
	// Default: "x-api-key"
	KeyHeader string `mapstructure:"key_header"`

	// CacheTTL is the duration to cache Key Service responses.
	// Must be a positive duration.
	// Default: 5m
	CacheTTL time.Duration `mapstructure:"cache_ttl"`

	// KeyServiceTimeout is the HTTP request timeout for Key Service calls.
	// Must be a positive duration.
	// Default: 5s
	KeyServiceTimeout time.Duration `mapstructure:"key_service_timeout"`

	// DefaultPipelines contains the pipeline IDs to route to when a key
	// cannot be resolved or is missing.
	// Optional. If empty, unresolved telemetry is dropped with a warning.
	DefaultPipelines []pipeline.ID `mapstructure:"default_pipelines"`

	// DefaultExporterType is the exporter type to use when the Key Service
	// response does not include an exporter_type field.
	// Default: "clickzetta"
	DefaultExporterType string `mapstructure:"default_exporter_type"`

	// ExporterIdleTimeout is the duration after which an unused dynamic
	// exporter is shut down and evicted from the cache.
	// Default: 30m
	ExporterIdleTimeout time.Duration `mapstructure:"exporter_idle_timeout"`

	// BatchSize is the number of records (log records / data points / spans)
	// to accumulate per API key before flushing to the downstream exporter.
	// This provides per-key batching without losing the API key context.
	// Set to 0 to disable buffering (forward immediately).
	// Default: 0 (disabled)
	BatchSize int `mapstructure:"batch_size"`

	// FlushInterval is the maximum time to wait before flushing a non-empty
	// per-key buffer, even if BatchSize has not been reached.
	// Only used when BatchSize > 0.
	// Default: 5s
	FlushInterval time.Duration `mapstructure:"flush_interval"`
}

// Validate checks the configuration for required fields and constraints.
func (c *Config) Validate() error {
	if c.KeyServiceURL == "" {
		return errEmptyKeyServiceURL
	}
	if !strings.Contains(c.KeyServiceURL, "{api_key}") {
		return errMissingAPIKeyPlaceholder
	}
	if c.CacheTTL <= 0 {
		return errNonPositiveCacheTTL
	}
	if c.KeyServiceTimeout <= 0 {
		return errNonPositiveServiceTimeout
	}
	if c.BatchSize > 0 && c.FlushInterval <= 0 {
		return errNonPositiveFlushInterval
	}
	return nil
}
