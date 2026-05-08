// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"

import "fmt"

// CreateLogsTableSQL returns the DDL for the logs table.
func CreateLogsTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	timestamp TIMESTAMP_LTZ,
	traceid STRING,
	spanid STRING,
	traceflags INT,
	severitytext STRING,
	severitynumber INT,
	servicename STRING,
	body STRING,
	resourceschemaurl STRING,
	resourceattributes MAP<STRING, STRING>,
	scopeschemaurl STRING,
	scopename STRING,
	scopeversion STRING,
	scopeattributes MAP<STRING, STRING>,
	logattributes MAP<STRING, STRING>,
	INDEX %s_idx_trace_id (traceid) BLOOMFILTER,
	INDEX %s_idx_span_id (spanid) BLOOMFILTER,
	INDEX %s_idx_service_name (servicename) INVERTED PROPERTIES('analyzer'='keyword'),
	INDEX %s_idx_severity_text (severitytext) INVERTED PROPERTIES('analyzer'='keyword'),
	INDEX %s_idx_body (body) INVERTED PROPERTIES('analyzer'='english')
) PARTITIONED BY (days(timestamp))`, tableName, tableName, tableName, tableName, tableName, tableName)
}

// CreateTracesTableSQL returns the DDL for the traces table.
func CreateTracesTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	timestamp TIMESTAMP_LTZ,
	traceid STRING,
	spanid STRING,
	parentspanid STRING,
	tracestate STRING,
	spanname STRING,
	spankind STRING,
	servicename STRING,
	resourceattributes MAP<STRING, STRING>,
	scopename STRING,
	scopeversion STRING,
	spanattributes MAP<STRING, STRING>,
	duration BIGINT,
	statuscode STRING,
	statusmessage STRING,
	events STRING,
	links STRING,
	INDEX %s_idx_trace_id (traceid) BLOOMFILTER,
	INDEX %s_idx_span_id (spanid) BLOOMFILTER,
	INDEX %s_idx_parent_span_id (parentspanid) BLOOMFILTER,
	INDEX %s_idx_service_name (servicename) INVERTED PROPERTIES('analyzer'='keyword'),
	INDEX %s_idx_span_name (spanname) INVERTED PROPERTIES('analyzer'='keyword')
) PARTITIONED BY (days(timestamp))`, tableName, tableName, tableName, tableName, tableName, tableName)
}

// metricsCommonColumns returns the shared column definitions for all metrics tables.
func metricsCommonColumns(tableName string) string {
	return fmt.Sprintf(`resourceattributes MAP<STRING, STRING>,
	resourceschemaurl STRING,
	scopename STRING,
	scopeversion STRING,
	scopeattributes MAP<STRING, STRING>,
	scopedroppedattrcount INT,
	scopeschemaurl STRING,
	servicename STRING,
	metricname STRING,
	metricdescription STRING,
	metricunit STRING,
	attributes MAP<STRING, STRING>,
	starttimeunix TIMESTAMP_LTZ,
	timeunix TIMESTAMP_LTZ,
	flags INT,
	INDEX %s_idx_service_name (servicename) INVERTED PROPERTIES('analyzer'='keyword'),
	INDEX %s_idx_metric_name (metricname) INVERTED PROPERTIES('analyzer'='keyword')`, tableName, tableName)
}

// CreateMetricsGaugeTableSQL returns the DDL for the gauge metrics table.
func CreateMetricsGaugeTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	%s,
	value DOUBLE,
	exemplars STRING
) PARTITIONED BY (days(timeunix))`, tableName, metricsCommonColumns(tableName))
}

// CreateMetricsSumTableSQL returns the DDL for the sum metrics table.
func CreateMetricsSumTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	%s,
	value DOUBLE,
	aggregationtemporality INT,
	ismonotonic BOOLEAN,
	exemplars STRING
) PARTITIONED BY (days(timeunix))`, tableName, metricsCommonColumns(tableName))
}

// CreateMetricsHistogramTableSQL returns the DDL for the histogram metrics table.
func CreateMetricsHistogramTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	%s,
	count BIGINT,
	sum DOUBLE,
	bucketcounts STRING,
	explicitbounds STRING,
	min DOUBLE,
	max DOUBLE,
	aggregationtemporality INT,
	exemplars STRING
) PARTITIONED BY (days(timeunix))`, tableName, metricsCommonColumns(tableName))
}

// CreateMetricsSummaryTableSQL returns the DDL for the summary metrics table.
func CreateMetricsSummaryTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	%s,
	count BIGINT,
	sum DOUBLE,
	valueatquantiles STRING
) PARTITIONED BY (days(timeunix))`, tableName, metricsCommonColumns(tableName))
}

// CreateMetricsExpHistogramTableSQL returns the DDL for the exponential histogram metrics table.
func CreateMetricsExpHistogramTableSQL(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	%s,
	count BIGINT,
	sum DOUBLE,
	scale INT,
	zerocount BIGINT,
	positiveoffset INT,
	positivebucketcounts STRING,
	negativeoffset INT,
	negativebucketcounts STRING,
	min DOUBLE,
	max DOUBLE,
	aggregationtemporality INT,
	exemplars STRING
) PARTITIONED BY (days(timeunix))`, tableName, metricsCommonColumns(tableName))
}
