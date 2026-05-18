// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/collector/client"
	"go.opentelemetry.io/collector/consumer/consumererror"
)

// extractAPIKey reads the API key from Client_Metadata in the context.
// Returns the trimmed key or a permanent error if missing/empty.
func extractAPIKey(ctx context.Context, headerName string) (string, error) {
	cl := client.FromContext(ctx)
	values := cl.Metadata.Get(headerName)
	if len(values) == 0 {
		return "", consumererror.NewPermanent(errors.New("API key header " + headerName + " not found in request metadata"))
	}
	key := strings.TrimSpace(values[0])
	if key == "" {
		return "", consumererror.NewPermanent(errors.New("API key header " + headerName + " is empty after trimming"))
	}
	return key, nil
}

// maskAPIKey returns the first 8 characters of the key followed by "..."
// For keys 8 characters or shorter, returns the full key followed by "..."
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return key + "..."
	}
	return key[:8] + "..."
}
