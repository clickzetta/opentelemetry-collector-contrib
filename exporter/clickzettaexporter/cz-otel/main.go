// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickzettaexporter/cz-otel/internal/cli"
)

// version and collectorVersion are set via -ldflags at build time.
var (
	version          = "dev"
	collectorVersion = "unknown"
)

func main() {
	cli.SetVersionInfo(version, collectorVersion)
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
