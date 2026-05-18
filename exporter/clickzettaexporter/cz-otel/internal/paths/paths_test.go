// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package paths

import (
	"runtime"
	"strings"
	"testing"
)

func TestBaseDirNonEmpty(t *testing.T) {
	dir := BaseDir()
	if dir == "" {
		t.Fatal("BaseDir() returned empty string")
	}
}

func TestBaseDirPlatformConvention(t *testing.T) {
	dir := BaseDir()
	if runtime.GOOS == "windows" {
		if !strings.Contains(dir, "cz-otel") {
			t.Errorf("BaseDir() on Windows should contain 'cz-otel', got %q", dir)
		}
	} else {
		if !strings.Contains(dir, ".cz-otel") {
			t.Errorf("BaseDir() on Unix should contain '.cz-otel', got %q", dir)
		}
	}
}

func TestCollectorBinaryPathEndsWithExpectedName(t *testing.T) {
	path := CollectorBinaryPath()
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(path, "otelcol-clickzetta.exe") {
			t.Errorf("CollectorBinaryPath() on Windows should end with 'otelcol-clickzetta.exe', got %q", path)
		}
	} else {
		if !strings.HasSuffix(path, "otelcol-clickzetta") {
			t.Errorf("CollectorBinaryPath() on Unix should end with 'otelcol-clickzetta', got %q", path)
		}
	}
}

func TestAllPathsUnderBaseDir(t *testing.T) {
	base := BaseDir()
	paths := []struct {
		name string
		path string
	}{
		{"ConfigFilePath", ConfigFilePath()},
		{"CollectorConfigPath", CollectorConfigPath()},
		{"CollectorBinaryPath", CollectorBinaryPath()},
		{"PIDFilePath", PIDFilePath()},
		{"LogFilePath", LogFilePath()},
		{"BinDir", BinDir()},
		{"LogsDir", LogsDir()},
	}

	for _, p := range paths {
		if !strings.HasPrefix(p.path, base) {
			t.Errorf("%s = %q does not start with BaseDir %q", p.name, p.path, base)
		}
	}
}

func TestConfigFilePathEndsWithConfigYaml(t *testing.T) {
	path := ConfigFilePath()
	if !strings.HasSuffix(path, "config.yaml") {
		t.Errorf("ConfigFilePath() should end with 'config.yaml', got %q", path)
	}
}

func TestCollectorConfigPathEndsWithExpectedName(t *testing.T) {
	path := CollectorConfigPath()
	if !strings.HasSuffix(path, "otel-collector-config.yaml") {
		t.Errorf("CollectorConfigPath() should end with 'otel-collector-config.yaml', got %q", path)
	}
}

func TestPIDFilePathEndsWithExpectedName(t *testing.T) {
	path := PIDFilePath()
	if !strings.HasSuffix(path, "collector.pid") {
		t.Errorf("PIDFilePath() should end with 'collector.pid', got %q", path)
	}
}

func TestLogFilePathEndsWithExpectedName(t *testing.T) {
	path := LogFilePath()
	if !strings.HasSuffix(path, "collector.log") {
		t.Errorf("LogFilePath() should end with 'collector.log', got %q", path)
	}
}
