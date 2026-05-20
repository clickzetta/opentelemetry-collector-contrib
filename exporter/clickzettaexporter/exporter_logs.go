// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"fmt"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

type logsExporter struct {
	baseExporter
}

func newLogsExporter(logger *zap.Logger, cfg *Config) *logsExporter {
	return &logsExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushLogsData ignores ctx because the goclickzetta BulkLoad API (v0.0.16) does not accept context.
func (e *logsExporter) pushLogsData(_ context.Context, ld plog.Logs) error {
	stream, err := e.conn.CreateBulkloadStream(goclickzetta.BulkloadOptions{
		Table:     e.cfg.LogsTableName,
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
	e.logger.Info("Successfully wrote logs to ClickZetta", zap.Int("log_records", ld.LogRecordCount()))
	return nil
}
