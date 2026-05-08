// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"encoding/json"
	"fmt"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

type metricsExporter struct {
	baseExporter
}

func newMetricsExporter(logger *zap.Logger, cfg *Config) *metricsExporter {
	return &metricsExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushMetricsData ignores ctx because the goclickzetta BulkLoad API (v0.0.16) does not accept context.
func (e *metricsExporter) pushMetricsData(_ context.Context, md pmetric.Metrics) error {
	// Group data points by metric type, then write each group to its table.
	type tableRows struct {
		table string
		rows  []map[string]interface{}
	}
	groups := map[pmetric.MetricType]*tableRows{
		pmetric.MetricTypeGauge:                {table: e.cfg.MetricsTableName + "_gauge"},
		pmetric.MetricTypeSum:                  {table: e.cfg.MetricsTableName + "_sum"},
		pmetric.MetricTypeHistogram:            {table: e.cfg.MetricsTableName + "_histogram"},
		pmetric.MetricTypeSummary:              {table: e.cfg.MetricsTableName + "_summary"},
		pmetric.MetricTypeExponentialHistogram: {table: e.cfg.MetricsTableName + "_exp_histogram"},
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
				common := func(attrs pcommon.Map, startTime, time pcommon.Timestamp, flags uint32) map[string]interface{} {
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
						"timeunix":              time.AsTime().Format("2006-01-02 15:04:05.999999999"),
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
	var totalRows int
	for _, g := range groups {
		if len(g.rows) == 0 {
			continue
		}
		if err := e.writeBulkload(g.table, g.rows); err != nil {
			return err
		}
		totalRows += len(g.rows)
	}
	e.logger.Info("Successfully wrote metrics to ClickZetta", zap.Int("data_points", totalRows))
	return nil
}

func (e *metricsExporter) writeBulkload(table string, rows []map[string]interface{}) error {
	stream, err := e.conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
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
			return fmt.Errorf("failed to write metric row to %s: %w", table, err)
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

type exemplarJSON struct {
	FilteredAttributes map[string]string `json:"FilteredAttributes"`
	Timestamp          string            `json:"Timestamp"`
	Value              float64           `json:"Value"`
	SpanId             string            `json:"SpanId"`
	TraceId            string            `json:"TraceId"`
}

func exemplarsJSON(exemplars pmetric.ExemplarSlice) string {
	if exemplars.Len() == 0 {
		return "[]"
	}
	result := make([]exemplarJSON, 0, exemplars.Len())
	for i := 0; i < exemplars.Len(); i++ {
		ex := exemplars.At(i)
		attrs := make(map[string]string, ex.FilteredAttributes().Len())
		ex.FilteredAttributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
		result = append(result, exemplarJSON{
			FilteredAttributes: attrs,
			Timestamp:          ex.Timestamp().AsTime().Format("2006-01-02 15:04:05.999999999"),
			Value:              ex.DoubleValue(),
			SpanId:             ex.SpanID().String(),
			TraceId:            ex.TraceID().String(),
		})
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "[]"
	}
	return string(b)
}

type quantileJSON struct {
	Quantile float64 `json:"Quantile"`
	Value    float64 `json:"Value"`
}

func quantilesJSON(qvs pmetric.SummaryDataPointValueAtQuantileSlice) string {
	if qvs.Len() == 0 {
		return "[]"
	}
	result := make([]quantileJSON, 0, qvs.Len())
	for i := 0; i < qvs.Len(); i++ {
		qv := qvs.At(i)
		result = append(result, quantileJSON{Quantile: qv.Quantile(), Value: qv.Value()})
	}
	b, err := json.Marshal(result)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func toJSONArray(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}
