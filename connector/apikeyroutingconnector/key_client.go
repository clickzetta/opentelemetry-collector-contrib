// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// errPermanentKeyInvalid indicates the API key is definitively invalid (401/404).
// Callers should not retry and should route to the default pipeline.
var errPermanentKeyInvalid = errors.New("API key is invalid (permanent)")

// retryableError wraps an error to indicate the operation may succeed on retry.
// This covers 5xx responses, timeouts, network errors, and JSON parse failures.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("retryable: %v", e.err)
}

func (e *retryableError) Unwrap() error {
	return e.err
}

// isRetryable checks whether an error is a retryable error.
func isRetryable(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

// KeyServiceResponse represents the JSON response from the Key Service.
// Format:
//
//	{
//	  "id": 0,
//	  "name": "tenant-a",
//	  "exporters": [
//	    { "exporter_id": "0", "exporter_type": "clickzetta", "exporter_config": {...} }
//	  ]
//	}
type KeyServiceResponse struct {
	// ID is the numeric tenant identifier.
	ID int `json:"id"`

	// Name is the tenant name, used as the logical pipeline identifier.
	Name string `json:"name"`

	// Exporters is the list of exporter targets for this tenant.
	Exporters []ExporterTarget `json:"exporters"`
}

// ExporterTarget represents a single exporter target with its config.
type ExporterTarget struct {
	ExporterID     string         `json:"exporter_id"`
	ExporterType   string         `json:"exporter_type"`
	ExporterConfig map[string]any `json:"exporter_config"`
}

// KeyServiceClient communicates with the external Key Service API.
type KeyServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKeyServiceClient creates a client with the given base URL and timeout.
func NewKeyServiceClient(baseURL string, timeout time.Duration) *KeyServiceClient {
	return &KeyServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// parseRouteEntry converts the Key Service response into a RouteEntry slice.
// Each exporter in the response becomes a RouteEntry.
// The tenant name is used as the PipelineID for all entries.
// When exporter_type is empty in any entry, defaultExporterType is used.
func parseRouteEntry(resp *KeyServiceResponse, defaultExporterType string) ([]*RouteEntry, error) {
	if len(resp.Exporters) == 0 {
		return nil, fmt.Errorf("key service response has no exporters for tenant %q", resp.Name)
	}

	entries := make([]*RouteEntry, 0, len(resp.Exporters))
	for _, e := range resp.Exporters {
		exporterType := e.ExporterType
		if exporterType == "" {
			exporterType = defaultExporterType
		}
		entries = append(entries, &RouteEntry{
			PipelineID:     resp.Name,
			ExporterID:     e.ExporterID,
			ExporterType:   exporterType,
			ExporterConfig: e.ExporterConfig,
		})
	}
	return entries, nil
}

// Resolve fetches the pipeline mapping for the given API key.
// Returns the parsed response, or an error categorized as:
//   - permanent (401/404): key is invalid, wrapped with errPermanentKeyInvalid
//   - retryable (5xx/timeout/network/parse error): service unavailable, wrapped as retryableError
func (c *KeyServiceClient) Resolve(ctx context.Context, apiKey string) (*KeyServiceResponse, error) {
	url := strings.ReplaceAll(c.baseURL, "{api_key}", apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &retryableError{err: fmt.Errorf("failed to create request: %w", err)}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network errors and timeouts are retryable.
		return nil, &retryableError{err: fmt.Errorf("key service request failed: %w", err)}
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, &retryableError{err: fmt.Errorf("failed to read response body: %w", err)}
		}

		var result KeyServiceResponse
		if err := json.Unmarshal(body, &result); err != nil {
			// JSON parse failure is treated as a service error (retryable).
			return nil, &retryableError{err: fmt.Errorf("failed to parse key service response: %w", err)}
		}

		return &result, nil

	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound:
		// Permanent error: key is invalid.
		return nil, fmt.Errorf("%w: key service returned %d", errPermanentKeyInvalid, resp.StatusCode)

	default:
		// 5xx and any other unexpected status codes are retryable.
		return nil, &retryableError{err: fmt.Errorf("key service returned unexpected status %d", resp.StatusCode)}
	}
}
