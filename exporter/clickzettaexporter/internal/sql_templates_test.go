// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateLogsTableSQL(t *testing.T) {
	sql := CreateLogsTableSQL("otel_logs")
	assert.Contains(t, sql, "otel_logs")
	assert.Contains(t, sql, "timestamp TIMESTAMP_LTZ")
	assert.Contains(t, sql, "traceid STRING")
	assert.Contains(t, sql, "logattributes MAP<STRING, STRING>")
	assert.Contains(t, sql, "PARTITIONED BY (days(timestamp))")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(sql), "CREATE TABLE IF NOT EXISTS"))
}

func TestCreateTracesTableSQL(t *testing.T) {
	sql := CreateTracesTableSQL("otel_traces")
	assert.Contains(t, sql, "otel_traces")
	assert.Contains(t, sql, "spanid STRING")
	assert.Contains(t, sql, "duration BIGINT")
	assert.Contains(t, sql, "events STRING")
	assert.Contains(t, sql, "PARTITIONED BY (days(timestamp))")
}

func TestCreateMetricsGaugeTableSQL(t *testing.T) {
	sql := CreateMetricsGaugeTableSQL("otel_metrics_gauge")
	assert.Contains(t, sql, "otel_metrics_gauge")
	assert.Contains(t, sql, "value DOUBLE")
	assert.Contains(t, sql, "exemplars STRING")
	assert.Contains(t, sql, "PARTITIONED BY (days(timeunix))")
}

func TestCreateMetricsSumTableSQL(t *testing.T) {
	sql := CreateMetricsSumTableSQL("otel_metrics_sum")
	assert.Contains(t, sql, "otel_metrics_sum")
	assert.Contains(t, sql, "ismonotonic BOOLEAN")
	assert.Contains(t, sql, "aggregationtemporality INT")
}

func TestCreateMetricsHistogramTableSQL(t *testing.T) {
	sql := CreateMetricsHistogramTableSQL("otel_metrics_histogram")
	assert.Contains(t, sql, "otel_metrics_histogram")
	assert.Contains(t, sql, "bucketcounts STRING")
	assert.Contains(t, sql, "explicitbounds STRING")
	assert.Contains(t, sql, "count BIGINT")
}

func TestCreateMetricsSummaryTableSQL(t *testing.T) {
	sql := CreateMetricsSummaryTableSQL("otel_metrics_summary")
	assert.Contains(t, sql, "otel_metrics_summary")
	assert.Contains(t, sql, "valueatquantiles STRING")
}

func TestCreateMetricsExpHistogramTableSQL(t *testing.T) {
	sql := CreateMetricsExpHistogramTableSQL("otel_metrics_exp_histogram")
	assert.Contains(t, sql, "otel_metrics_exp_histogram")
	assert.Contains(t, sql, "positivebucketcounts STRING")
	assert.Contains(t, sql, "negativebucketcounts STRING")
	assert.Contains(t, sql, "zerocount BIGINT")
}
