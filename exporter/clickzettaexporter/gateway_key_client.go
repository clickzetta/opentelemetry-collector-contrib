// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/collector/consumer/consumererror"
)

// TenantConfig holds per-tenant ClickZetta connection parameters
// resolved from the Key Service.
type TenantConfig struct {
	TenantName     string `json:"tenant_name"`
	Service        string `json:"service"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Workspace      string `json:"workspace"`
	Instance       string `json:"instance"`
	VirtualCluster string `json:"virtual_cluster"`
	Schema         string `json:"schema"`
}

var (
	errInvalidKey         = errors.New("invalid API key")
	errServiceUnavailable = errors.New("key service unavailable")
)

// KeyServiceClient communicates with the external Key Service API.
type KeyServiceClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKeyServiceClient creates a new KeyServiceClient with a 5-second timeout.
func NewKeyServiceClient(baseURL string) *KeyServiceClient {
	return &KeyServiceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Resolve fetches the TenantConfig for the given API key from the Key Service.
func (c *KeyServiceClient) Resolve(ctx context.Context, apiKey string) (*TenantConfig, error) {
	url := c.baseURL + "/api/keys/" + apiKey

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Timeout or network error — retryable
		return nil, fmt.Errorf("%w: %v", errServiceUnavailable, err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to read response body: %v", errServiceUnavailable, err)
		}
		var config TenantConfig
		if err := json.Unmarshal(body, &config); err != nil {
			return nil, fmt.Errorf("%w: failed to parse response: %v", errServiceUnavailable, err)
		}
		return &config, nil

	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound:
		return nil, consumererror.NewPermanent(errInvalidKey)

	default:
		// 5xx or other unexpected status — retryable
		return nil, fmt.Errorf("%w: HTTP %d", errServiceUnavailable, resp.StatusCode)
	}
}
