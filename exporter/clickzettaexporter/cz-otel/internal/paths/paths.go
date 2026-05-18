// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	dirName             = "cz-otel"
	configFileName      = "config.yaml"
	collectorConfigName = "otel-collector-config.yaml"
	pidFileName         = "collector.pid"
	logFileName         = "collector.log"
	binDirName          = "bin"
	logsDirName         = "logs"
	collectorBinaryName = "otelcol-clickzetta"
)

// BaseDir returns the platform-specific base directory for cz-otel.
// On Unix (macOS/Linux): ~/.cz-otel/
// On Windows: %APPDATA%\cz-otel\
func BaseDir() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		return filepath.Join(appData, dirName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "."+dirName)
}

// ConfigFilePath returns the path to the CLI config file (config.yaml).
func ConfigFilePath() string {
	return filepath.Join(BaseDir(), configFileName)
}

// CollectorConfigPath returns the path to the collector config file (otel-collector-config.yaml).
func CollectorConfigPath() string {
	return filepath.Join(BaseDir(), collectorConfigName)
}

// CollectorBinaryPath returns the path to the collector binary.
func CollectorBinaryPath() string {
	binary := collectorBinaryName
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	return filepath.Join(BinDir(), binary)
}

// PIDFilePath returns the path to the collector PID file.
func PIDFilePath() string {
	return filepath.Join(BaseDir(), pidFileName)
}

// LogFilePath returns the path to the collector log file.
func LogFilePath() string {
	return filepath.Join(LogsDir(), logFileName)
}

// BinDir returns the path to the bin/ directory.
func BinDir() string {
	return filepath.Join(BaseDir(), binDirName)
}

// LogsDir returns the path to the logs/ directory.
func LogsDir() string {
	return filepath.Join(BaseDir(), logsDirName)
}
