// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"encoding/json"
	"fmt"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

// writeLogs writes log data to ClickZetta via the given bulkload connection.
func writeLogs(conn *goclickzetta.ClickzettaConn, cfg *Config, ld plog.Logs) error {
	stream, err := conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
		Table:     cfg.LogsTableName,
		Operation: goclickzetta.APPEND,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulkload stream for logs: %w", err)
	}

	writer, err := stream.OpenWriter(0)
	if err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to open writer for logs: %w", err)
	}

	rsLogs := ld.ResourceLogs()
	for i := 0; i < rsLogs.Len(); i++ {
		rl := rsLogs.At(i)
		res := rl.Resource()
		resAttr := res.Attributes()
		serviceName := internal.GetServiceName(resAttr)
		resAttrMap := internal.AttributesToMap(resAttr)
		resURL := rl.SchemaUrl()

		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			scope := sl.Scope()
			scopeURL := sl.SchemaUrl()
			scopeAttrMap := internal.AttributesToMap(scope.Attributes())

			for k := 0; k < sl.LogRecords().Len(); k++ {
				r := sl.LogRecords().At(k)
				ts := r.Timestamp()
				if ts == 0 {
					ts = r.ObservedTimestamp()
				}

				row := writer.CreateRow()
				row.ColumnNameValues["timestamp"] = ts.AsTime().Format("2006-01-02 15:04:05.999999999")
				row.ColumnNameValues["traceid"] = r.TraceID().String()
				row.ColumnNameValues["spanid"] = r.SpanID().String()
				row.ColumnNameValues["traceflags"] = int32(r.Flags())
				row.ColumnNameValues["severitytext"] = r.SeverityText()
				row.ColumnNameValues["severitynumber"] = int32(r.SeverityNumber())
				row.ColumnNameValues["servicename"] = serviceName
				row.ColumnNameValues["body"] = r.Body().AsString()
				row.ColumnNameValues["resourceschemaurl"] = resURL
				row.ColumnNameValues["resourceattributes"] = resAttrMap
				row.ColumnNameValues["scopeschemaurl"] = scopeURL
				row.ColumnNameValues["scopename"] = scope.Name()
				row.ColumnNameValues["scopeversion"] = scope.Version()
				row.ColumnNameValues["scopeattributes"] = scopeAttrMap
				row.ColumnNameValues["logattributes"] = internal.AttributesToMap(r.Attributes())

				if err := writer.WriteRow(row); err != nil {
					_ = stream.Abort()
					return fmt.Errorf("failed to write log row: %w", err)
				}
			}
		}
	}

	if err := writer.Close(); err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to close logs writer: %w", err)
	}
	if err := stream.Close(); err != nil {
		return fmt.Errorf("failed to close logs stream: %w", err)
	}
	return nil
}

// writeTraces writes trace data to ClickZetta via the given bulkload connection.
func writeTraces(conn *goclickzetta.ClickzettaConn, cfg *Config, td ptrace.Traces) error {
	stream, err := conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
		Table:     cfg.TracesTableName,
		Operation: goclickzetta.APPEND,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulkload stream for traces: %w", err)
	}

	writer, err := stream.OpenWriter(0)
	if err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to open writer for traces: %w", err)
	}

	rsSpans := td.ResourceSpans()
	for i := 0; i < rsSpans.Len(); i++ {
		rs := rsSpans.At(i)
		res := rs.Resource()
		resAttr := res.Attributes()
		serviceName := internal.GetServiceName(resAttr)
		resAttrMap := internal.AttributesToMap(resAttr)

		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			ss := rs.ScopeSpans().At(j)
			scope := ss.Scope()

			for k := 0; k < ss.Spans().Len(); k++ {
				span := ss.Spans().At(k)
				durationNanos := int64(span.EndTimestamp() - span.StartTimestamp())

				eventsJSON, err := json.Marshal(convertEvents(span.Events()))
				if err != nil {
					eventsJSON = []byte("[]")
				}
				linksJSON, err := json.Marshal(convertLinks(span.Links()))
				if err != nil {
					linksJSON = []byte("[]")
				}

				row := writer.CreateRow()
				row.ColumnNameValues["timestamp"] = span.StartTimestamp().AsTime().Format("2006-01-02 15:04:05.999999999")
				row.ColumnNameValues["traceid"] = span.TraceID().String()
				row.ColumnNameValues["spanid"] = span.SpanID().String()
				row.ColumnNameValues["parentspanid"] = span.ParentSpanID().String()
				row.ColumnNameValues["tracestate"] = span.TraceState().AsRaw()
				row.ColumnNameValues["spanname"] = span.Name()
				row.ColumnNameValues["spankind"] = span.Kind().String()
				row.ColumnNameValues["servicename"] = serviceName
				row.ColumnNameValues["resourceattributes"] = resAttrMap
				row.ColumnNameValues["scopename"] = scope.Name()
				row.ColumnNameValues["scopeversion"] = scope.Version()
				row.ColumnNameValues["spanattributes"] = internal.AttributesToMap(span.Attributes())
				row.ColumnNameValues["duration"] = durationNanos
				row.ColumnNameValues["statuscode"] = span.Status().Code().String()
				row.ColumnNameValues["statusmessage"] = span.Status().Message()
				row.ColumnNameValues["events"] = string(eventsJSON)
				row.ColumnNameValues["links"] = string(linksJSON)

				if err := writer.WriteRow(row); err != nil {
					_ = stream.Abort()
					return fmt.Errorf("failed to write trace row: %w", err)
				}
			}
		}
	}

	if err := writer.Close(); err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to close traces writer: %w", err)
	}
	if err := stream.Close(); err != nil {
		return fmt.Errorf("failed to close traces stream: %w", err)
	}
	return nil
}

// writeMetrics writes metric data to ClickZetta via the given bulkload connection.
func writeMetrics(conn *goclickzetta.ClickzettaConn, cfg *Config, md pmetric.Metrics) error {
	type tableRows struct {
		table string
		rows  []map[string]interface{}
	}
	groups := map[pmetric.MetricType]*tableRows{
		pmetric.MetricTypeGauge:                {table: cfg.MetricsTableName + "_gauge"},
		pmetric.MetricTypeSum:                  {table: cfg.MetricsTableName + "_sum"},
		pmetric.MetricTypeHistogram:            {table: cfg.MetricsTableName + "_histogram"},
		pmetric.MetricTypeSummary:              {table: cfg.MetricsTableName + "_summary"},
		pmetric.MetricTypeExponentialHistogram: {table: cfg.MetricsTableName + "_exp_histogram"},
	}

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rm := md.ResourceMetrics().At(i)
		resAttr := rm.Resource().Attributes()
		serviceName := internal.GetServiceName(resAttr)
		resAttrMap := internal.AttributesToMap(resAttr)
		resURL := rm.SchemaUrl()

		for j := 0; j < rm.ScopeMetrics().Len(); j++ {
			sm := rm.ScopeMetrics().At(j)
			scope := sm.Scope()
			scopeURL := sm.SchemaUrl()
			scopeAttrMap := internal.AttributesToMap(scope.Attributes())

			for k := 0; k < sm.Metrics().Len(); k++ {
				m := sm.Metrics().At(k)
				common := func(attrs pcommon.Map, startTime, ts pcommon.Timestamp, flags uint32) map[string]interface{} {
					return map[string]interface{}{
						"resourceattributes":    resAttrMap,
						"resourceschemaurl":     resURL,
						"scopename":             scope.Name(),
						"scopeversion":          scope.Version(),
						"scopeattributes":       scopeAttrMap,
						"scopedroppedattrcount": int32(scope.DroppedAttributesCount()),
						"scopeschemaurl":        scopeURL,
						"servicename":           serviceName,
						"metricname":            m.Name(),
						"metricdescription":     m.Description(),
						"metricunit":            m.Unit(),
						"attributes":            internal.AttributesToMap(attrs),
						"starttimeunix":         startTime.AsTime().Format("2006-01-02 15:04:05.999999999"),
						"timeunix":              ts.AsTime().Format("2006-01-02 15:04:05.999999999"),
						"flags":                 int32(flags),
					}
				}

				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dps := m.Gauge().DataPoints()
					for di := 0; di < dps.Len(); di++ {
						dp := dps.At(di)
						row := common(dp.Attributes(), dp.StartTimestamp(), dp.Timestamp(), uint32(dp.Flags()))
						if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
							row["value"] = float64(dp.IntValue())
						} else {
							row["value"] = dp.DoubleValue()
						}
						row["exemplars"] = exemplarsJSON(dp.Exemplars())
						groups[m.Type()].rows = append(groups[m.Type()].rows, row)
					}
				case pmetric.MetricTypeSum:
					dps := m.Sum().DataPoints()
					for di := 0; di < dps.Len(); di++ {
						dp := dps.At(di)
						row := common(dp.Attributes(), dp.StartTimestamp(), dp.Timestamp(), uint32(dp.Flags()))
						if dp.ValueType() == pmetric.NumberDataPointValueTypeInt {
							row["value"] = float64(dp.IntValue())
						} else {
							row["value"] = dp.DoubleValue()
						}
						row["aggregationtemporality"] = int32(m.Sum().AggregationTemporality())
						row["ismonotonic"] = m.Sum().IsMonotonic()
						row["exemplars"] = exemplarsJSON(dp.Exemplars())
						groups[m.Type()].rows = append(groups[m.Type()].rows, row)
					}
				case pmetric.MetricTypeHistogram:
					dps := m.Histogram().DataPoints()
					for di := 0; di < dps.Len(); di++ {
						dp := dps.At(di)
						row := common(dp.Attributes(), dp.StartTimestamp(), dp.Timestamp(), uint32(dp.Flags()))
						row["count"] = int64(dp.Count())
						row["sum"] = dp.Sum()
						row["bucketcounts"] = toJSONArray(dp.BucketCounts().AsRaw())
						row["explicitbounds"] = toJSONArray(dp.ExplicitBounds().AsRaw())
						row["min"] = dp.Min()
						row["max"] = dp.Max()
						row["aggregationtemporality"] = int32(m.Histogram().AggregationTemporality())
						row["exemplars"] = exemplarsJSON(dp.Exemplars())
						groups[m.Type()].rows = append(groups[m.Type()].rows, row)
					}
				case pmetric.MetricTypeSummary:
					dps := m.Summary().DataPoints()
					for di := 0; di < dps.Len(); di++ {
						dp := dps.At(di)
						row := common(dp.Attributes(), dp.StartTimestamp(), dp.Timestamp(), uint32(dp.Flags()))
						row["count"] = int64(dp.Count())
						row["sum"] = dp.Sum()
						row["valueatquantiles"] = quantilesJSON(dp.QuantileValues())
						groups[m.Type()].rows = append(groups[m.Type()].rows, row)
					}
				case pmetric.MetricTypeExponentialHistogram:
					dps := m.ExponentialHistogram().DataPoints()
					for di := 0; di < dps.Len(); di++ {
						dp := dps.At(di)
						row := common(dp.Attributes(), dp.StartTimestamp(), dp.Timestamp(), uint32(dp.Flags()))
						row["count"] = int64(dp.Count())
						row["sum"] = dp.Sum()
						row["scale"] = int32(dp.Scale())
						row["zerocount"] = int64(dp.ZeroCount())
						row["positiveoffset"] = int32(dp.Positive().Offset())
						row["positivebucketcounts"] = toJSONArray(dp.Positive().BucketCounts().AsRaw())
						row["negativeoffset"] = int32(dp.Negative().Offset())
						row["negativebucketcounts"] = toJSONArray(dp.Negative().BucketCounts().AsRaw())
						row["min"] = dp.Min()
						row["max"] = dp.Max()
						row["aggregationtemporality"] = int32(m.ExponentialHistogram().AggregationTemporality())
						row["exemplars"] = exemplarsJSON(dp.Exemplars())
						groups[m.Type()].rows = append(groups[m.Type()].rows, row)
					}
				}
			}
		}
	}

	// Write each group to its table via BulkLoad.
	for _, g := range groups {
		if len(g.rows) == 0 {
			continue
		}
		if err := writeBulkloadRows(conn, g.table, g.rows); err != nil {
			return err
		}
	}
	return nil
}

// writeBulkloadRows writes pre-built rows to a table using the BulkLoad stream pattern.
func writeBulkloadRows(conn *goclickzetta.ClickzettaConn, table string, rows []map[string]interface{}) error {
	stream, err := conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
		Table:     table,
		Operation: goclickzetta.APPEND,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulkload stream for %s: %w", table, err)
	}

	writer, err := stream.OpenWriter(0)
	if err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to open writer for %s: %w", table, err)
	}

	for _, rowData := range rows {
		row := writer.CreateRow()
		for k, v := range rowData {
			row.ColumnNameValues[k] = v
		}
		if err := writer.WriteRow(row); err != nil {
			_ = stream.Abort()
			return fmt.Errorf("failed to write row to %s: %w", table, err)
		}
	}

	if err := writer.Close(); err != nil {
		_ = stream.Abort()
		return fmt.Errorf("failed to close writer for %s: %w", table, err)
	}
	if err := stream.Close(); err != nil {
		return fmt.Errorf("failed to close stream for %s: %w", table, err)
	}
	return nil
}
