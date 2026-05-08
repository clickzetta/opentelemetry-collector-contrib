// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateTables creates all OTEL tables in ClickZetta if they don't exist.
func CreateTables(ctx context.Context, db *sql.DB, logsTable, tracesTable, metricsTablePrefix string) error {
	ddls := []string{
		CreateLogsTableSQL(logsTable),
		CreateTracesTableSQL(tracesTable),
		CreateMetricsGaugeTableSQL(metricsTablePrefix + "_gauge"),
		CreateMetricsSumTableSQL(metricsTablePrefix + "_sum"),
		CreateMetricsHistogramTableSQL(metricsTablePrefix + "_histogram"),
		CreateMetricsSummaryTableSQL(metricsTablePrefix + "_summary"),
		CreateMetricsExpHistogramTableSQL(metricsTablePrefix + "_exp_histogram"),
	}
	for _, ddl := range ddls {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}
	return nil
}
