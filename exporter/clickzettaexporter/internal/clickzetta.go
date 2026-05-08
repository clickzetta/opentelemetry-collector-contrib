// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/internal"

import (
	"database/sql"

	goclickzetta "github.com/clickzetta/goclickzetta"
)

// NewSQLDB opens a database/sql connection to ClickZetta for DDL operations.
func NewSQLDB(dsn string) (*sql.DB, error) {
	return sql.Open("clickzetta", dsn)
}

// NewBulkloadConn creates a ClickzettaConn for BulkLoad operations.
func NewBulkloadConn(dsn string) (*goclickzetta.ClickzettaConn, error) {
	d := goclickzetta.ClickzettaDriver{}
	conn, err := d.Open(dsn)
	if err != nil {
		return nil, err
	}
	return conn.(*goclickzetta.ClickzettaConn), nil
}
