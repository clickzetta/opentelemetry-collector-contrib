// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestGetServiceName_Present(t *testing.T) {
	m := pcommon.NewMap()
	m.PutStr("service.name", "my-service")
	assert.Equal(t, "my-service", GetServiceName(m))
}

func TestGetServiceName_Absent(t *testing.T) {
	m := pcommon.NewMap()
	assert.Equal(t, "", GetServiceName(m))
}

func TestAttributesToMap_StringValues(t *testing.T) {
	m := pcommon.NewMap()
	m.PutStr("key1", "val1")
	m.PutStr("key2", "val2")
	result := AttributesToMap(m)
	assert.Equal(t, "val1", result["key1"])
	assert.Equal(t, "val2", result["key2"])
}

func TestAttributesToMap_NonStringCoercedToString(t *testing.T) {
	m := pcommon.NewMap()
	m.PutInt("count", 42)
	m.PutBool("enabled", true)
	m.PutDouble("ratio", 3.14)
	result := AttributesToMap(m)
	// All values become strings per MAP<STRING,STRING> schema.
	assert.Equal(t, "42", result["count"])
	assert.Equal(t, "true", result["enabled"])
	assert.Equal(t, "3.14", result["ratio"])
}

func TestAttributesToMap_Empty(t *testing.T) {
	m := pcommon.NewMap()
	result := AttributesToMap(m)
	assert.Empty(t, result)
}
