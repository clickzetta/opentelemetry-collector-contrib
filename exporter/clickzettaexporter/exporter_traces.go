// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type tracesExporter struct {
	baseExporter
}

func newTracesExporter(logger *zap.Logger, cfg *Config) *tracesExporter {
	return &tracesExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushTraceData writes traces to ClickZetta using the shared writeTraces function.
func (e *tracesExporter) pushTraceData(_ context.Context, td ptrace.Traces) error {
	if err := writeTraces(e.conn, e.cfg, td); err != nil {
		return err
	}
	e.logger.Info("Successfully wrote traces to ClickZetta", zap.Int("spans", td.SpanCount()))
	return nil
}

type eventJSON struct {
	Timestamp  string            `json:"Timestamp"`
	Name       string            `json:"Name"`
	Attributes map[string]string `json:"Attributes"`
}

type linkJSON struct {
	TraceId    string            `json:"TraceId"`
	SpanId     string            `json:"SpanId"`
	TraceState string            `json:"TraceState"`
	Attributes map[string]string `json:"Attributes"`
}

func convertEvents(events ptrace.SpanEventSlice) []eventJSON {
	if events.Len() == 0 {
		return nil
	}
	result := make([]eventJSON, 0, events.Len())
	for i := 0; i < events.Len(); i++ {
		e := events.At(i)
		attrs := make(map[string]string, e.Attributes().Len())
		e.Attributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
		result = append(result, eventJSON{
			Timestamp:  e.Timestamp().AsTime().Format("2006-01-02 15:04:05.999999999"),
			Name:       e.Name(),
			Attributes: attrs,
		})
	}
	return result
}

func convertLinks(links ptrace.SpanLinkSlice) []linkJSON {
	if links.Len() == 0 {
		return nil
	}
	result := make([]linkJSON, 0, links.Len())
	for i := 0; i < links.Len(); i++ {
		l := links.At(i)
		attrs := make(map[string]string, l.Attributes().Len())
		l.Attributes().Range(func(k string, v pcommon.Value) bool {
			attrs[k] = v.AsString()
			return true
		})
		result = append(result, linkJSON{
			TraceId:    l.TraceID().String(),
			SpanId:     l.SpanID().String(),
			TraceState: l.TraceState().AsRaw(),
			Attributes: attrs,
		})
	}
	return result
}
