// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/config"
)

// parsedTemplate is the pre-parsed collector config template.
var parsedTemplate = template.Must(template.New("collector-config").Parse(collectorConfigTemplate))

// Generate produces the collector configuration YAML from the given config data.
// It validates that all required keys are present before generating.
// Returns the YAML bytes or an error if validation fails.
func Generate(data map[string]string) ([]byte, error) {
	errors := config.ValidateAll(data)
	if len(errors) > 0 {
		return nil, fmt.Errorf("configuration validation failed:\n  %s", strings.Join(errors, "\n  "))
	}

	// Apply defaults for optional fields not explicitly set.
	resolved := make(map[string]string)
	for _, key := range config.ValidKeys {
		if val, ok := data[key]; ok && val != "" {
			resolved[key] = val
		} else if def := config.DefaultValue(key); def != "" {
			resolved[key] = def
		}
	}

	yaml, err := renderTemplate(resolved)
	if err != nil {
		return nil, fmt.Errorf("failed to render collector config template: %w", err)
	}
	return yaml, nil
}

// renderTemplate executes the collector config template with the resolved config values.
func renderTemplate(data map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	if err := parsedTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
