// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter"

import (
	"context"
	"database/sql"
	"fmt"

	goclickzetta "github.com/clickzetta/goclickzetta"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"
)

type baseExporter struct {
	logger *zap.Logger
	cfg    *Config
	db     *sql.DB
	conn   *goclickzetta.ClickzettaConn
}

func newBaseExporter(logger *zap.Logger, cfg *Config) baseExporter {
	return baseExporter{logger: logger, cfg: cfg}
}

func (b *baseExporter) start(_ context.Context, _ component.Host) error {
	dsn := b.cfg.DSN()

	if b.cfg.CreateSchema {
		db, err := internal.NewSQLDB(dsn)
		if err != nil {
			b.logger.Warn("failed to open SQL connection for schema creation", zap.String("dsn", b.cfg.redactedDSN()), zap.Error(err))
			return fmt.Errorf("failed to open SQL connection for schema creation: %w", err)
		}
		b.db = db
		if err = internal.CreateTables(context.Background(), b.db, b.cfg.LogsTableName, b.cfg.TracesTableName, b.cfg.MetricsTableName); err != nil {
			b.logger.Warn("failed to create tables", zap.Error(err))
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}

	conn, err := internal.NewBulkloadConn(dsn)
	if err != nil {
		return fmt.Errorf("failed to open bulkload connection: %w", err)
	}
	b.conn = conn
	return nil
}

func (b *baseExporter) shutdown(_ context.Context) error {
	if b.db != nil {
		_ = b.db.Close()
	}
	if b.conn != nil {
		_ = b.conn.Close()
	}
	return nil
}
