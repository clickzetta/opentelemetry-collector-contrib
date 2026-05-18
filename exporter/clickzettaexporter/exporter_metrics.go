// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsExporter struct {
	baseExporter
}

func newMetricsExporter(logger *zap.Logger, cfg *Config) *metricsExporter {
	return &metricsExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushMetricsData writes metrics to ClickZetta using the shared writeMetrics function.
func (e *metricsExporter) pushMetricsData(_ context.Context, md pmetric.Metrics) error {
	if err := writeMetrics(e.conn, e.cfg, md); err != nil {
		return err
	}
	e.logger.Info("Successfully wrote metrics to ClickZetta", zap.Int("data_points", md.DataPointCount()))
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


