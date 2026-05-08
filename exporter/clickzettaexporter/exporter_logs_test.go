// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package clickzettaexporter

// Integration test: writes log records through the real logsExporter, then
// queries ClickZetta via database/sql to verify the rows landed correctly.
//
// Run with:
//
//	CLICKZETTA_SERVICE=... CLICKZETTA_USERNAME=... CLICKZETTA_PASSWORD=... \
//	  go test -v -run TestLogsExporter_Integration -timeout 120s

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

func integrationConfig(t *testing.T, logsTable string) *Config {
	t.Helper()
	service := os.Getenv("CLICKZETTA_SERVICE")
	if service == "" {
		t.Skip("CLICKZETTA_SERVICE not set, skipping integration test")
	}
	return &Config{
		Service:          service,
		Username:         os.Getenv("CLICKZETTA_USERNAME"),
		Password:         configopaque.String(os.Getenv("CLICKZETTA_PASSWORD")),
		Workspace:        os.Getenv("CLICKZETTA_WORKSPACE"),
		VirtualCluster:   os.Getenv("CLICKZETTA_VIRTUAL_CLUSTER"),
		Instance:         os.Getenv("CLICKZETTA_INSTANCE"),
		Schema:           "public",
		Protocol:         "https",
		LogsTableName:    logsTable,
		TracesTableName:  "otel_traces_test",
		MetricsTableName: "otel_metrics_test",
		CreateSchema:     true,
	}
}

func makeLogs(serviceName string, bodies []string) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	rl.Resource().Attributes().PutStr("service.name", serviceName)
	sl := rl.ScopeLogs().AppendEmpty()
	for _, body := range bodies {
		lr := sl.LogRecords().AppendEmpty()
		lr.Body().SetStr(body)
	}
	return ld
}

func TestLogsExporter_StartFailsWithBadDSN(t *testing.T) {
	cfg := &Config{
		Service:        "invalid-host",
		Username:       "u",
		Password:       "p",
		Workspace:      "ws",
		VirtualCluster: "vc",
		Instance:       "inst",
		Schema:         "public",
		Protocol:       "https",
		CreateSchema:   false,
	}
	exp := newLogsExporter(zap.NewNop(), cfg)
	err := exp.start(context.Background(), nil)
	assert.Error(t, err, "start() should return an error when the bulkload connection fails")
}

func TestLogsExporter_Integration(t *testing.T) {
	tableName := fmt.Sprintf("otel_logs_test_%d", time.Now().UnixMilli())
	cfg := integrationConfig(t, tableName)
	t.Logf("table: %s", tableName)

	exp := newLogsExporter(zap.NewExample(), cfg)
	require.NoError(t, exp.start(context.Background(), nil))
	require.NotNil(t, exp.conn)
	t.Cleanup(func() { _ = exp.shutdown(context.Background()) })

	bodies := []string{"msg-0", "msg-1", "msg-2"}
	require.NoError(t, exp.pushLogsData(context.Background(), makeLogs("svc-a", bodies)))
	t.Log("pushLogsData OK")

	// Poll up to 60s for the commit to become query-visible.
	db, err := sql.Open("clickzetta", cfg.DSN())
	require.NoError(t, err)
	defer db.Close()

	var count int
	for i := 0; i < 12; i++ {
		time.Sleep(5 * time.Second)
		_ = db.QueryRowContext(context.Background(),
			fmt.Sprintf("SELECT COUNT(1) FROM %s", tableName)).Scan(&count)
		t.Logf("poll %ds: count=%d", (i+1)*5, count)
		if count > 0 {
			break
		}
	}

	assert.Equal(t, len(bodies), count, "row count mismatch")
}
