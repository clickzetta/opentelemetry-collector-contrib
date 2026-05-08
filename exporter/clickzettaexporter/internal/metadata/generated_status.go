// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package metadata provides the component metadata for the ClickZetta exporter.
package metadata // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal/metadata"

import "go.opentelemetry.io/collector/component"

var (
	Type               = component.MustNewType("clickzetta")
	LogsStability      = component.StabilityLevelDevelopment
	TracesStability    = component.StabilityLevelDevelopment
	MetricsStability   = component.StabilityLevelDevelopment
)
