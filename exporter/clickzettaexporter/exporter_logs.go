// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsExporter struct {
	baseExporter
}

func newLogsExporter(logger *zap.Logger, cfg *Config) *logsExporter {
	return &logsExporter{baseExporter: newBaseExporter(logger, cfg)}
}

// pushLogsData writes logs to ClickZetta using the shared writeLogs function.
func (e *logsExporter) pushLogsData(_ context.Context, ld plog.Logs) error {
	if err := writeLogs(e.conn, e.cfg, ld); err != nil {
		return err
	}
	e.logger.Info("Successfully wrote logs to ClickZetta", zap.Int("log_records", ld.LogRecordCount()))
	return nil
}
