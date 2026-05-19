// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package apikeyroutingconnector // import "github.com/open-telemetry/opentelemetry-collector-contrib/connector/apikeyroutingconnector"

import (
	"context"
	"errors"
	"strings"

	"go.opentelemetry.io/collector/client"
)

var (
	errMissingKey = errors.New("API key header is missing from client metadata")
	errEmptyKey   = errors.New("API key header value is empty after trimming")
)

// extractAPIKey reads the API key from Client_Metadata in the context.
// Returns the trimmed key value.
// Returns ("", errMissingKey) if the header is absent.
// Returns ("", errEmptyKey) if the header value is empty after trimming.
func extractAPIKey(ctx context.Context, headerName string) (string, error) {
	cl := client.FromContext(ctx)
	values := cl.Metadata.Get(headerName)
	if len(values) == 0 {
		return "", errMissingKey
	}

	key := strings.TrimSpace(values[0])
	if key == "" {
		return "", errEmptyKey
	}

	return key, nil
}

// maskAPIKey returns the first 8 characters of the key followed by "..."
// If the key is shorter than 8 characters, the full key is returned followed by "..."
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return key + "..."
	}
	return key[:8] + "..."
}
