// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"encoding/json"
	"fmt"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

type tracesExporter struct {
	baseExporter
}

func newTracesExporter(logger *zap.Logger, cfg *Config) *tracesExporter {
	return &tracesExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushTraceData ignores ctx because the goclickzetta BulkLoad API (v0.0.16) does not accept context.
func (e *tracesExporter) pushTraceData(_ context.Context, td ptrace.Traces) error {
	stream, err := e.conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
		Table:     e.cfg.TracesTableName,
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
