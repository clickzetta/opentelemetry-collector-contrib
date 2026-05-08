// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	conventions "go.opentelemetry.io/otel/semconv/v1.38.0"
)

// GetServiceName extracts the service name from resource attributes.
func GetServiceName(resAttr pcommon.Map) string {
	if v, ok := resAttr.Get(string(conventions.ServiceNameKey)); ok {
		return v.AsString()
	}
	return ""
}

// AttributesToMap converts pcommon.Map to a map[string]interface{} for ClickZetta MAP columns.
// All values are coerced to strings because the schema uses MAP<STRING, STRING>.
func AttributesToMap(attributes pcommon.Map) map[string]interface{} {
	m := make(map[string]interface{}, attributes.Len())
	attributes.Range(func(k string, v pcommon.Value) bool {
		m[k] = v.AsString()
		return true
	})
	return m
}
