# cz-otel — ClickZetta OpenTelemetry Collector CLI

A command-line tool for configuring and running a custom OpenTelemetry Collector with the ClickZetta exporter plugin. It handles configuration management, collector config generation, and daemon lifecycle — so you can get telemetry data flowing into ClickZetta Lakehouse with a few simple commands.

## Installation

**One-line install (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/clickzetta/exporter/clickzettaexporter/cz-otel/scripts/install.sh | bash
```

Or install a specific version:

```bash
CZ_OTEL_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/clickzetta/opentelemetry-collector-contrib/clickzetta/exporter/clickzettaexporter/cz-otel/scripts/install.sh | bash
```

**Manual install:**

1. Download the platform-specific package from the releases page.
2. Extract: `tar -xzf cz-otel-<version>-<os>-<arch>.tar.gz`
3. Copy binaries to `~/.cz-otel/bin/` and add to PATH.

The installer does not require root privileges.

## Quick Start

```bash
# 1. Interactive setup — prompts for service URL, credentials, workspace, etc.
cz-otel config init

# 2. Start the collector as a background daemon
cz-otel start

# 3. Check collector status
cz-otel status
```

The collector listens for OTLP data on gRPC (port 4317) and HTTP (port 4318) by default.

## Command Reference

| Command                         | Description                                                |
|---------------------------------|------------------------------------------------------------|
| `cz-otel config init`          | Interactive setup wizard for all configuration fields      |
| `cz-otel config set <key> <value>` | Set a single configuration value                       |
| `cz-otel config get <key>`     | Retrieve a stored configuration value                      |
| `cz-otel config list`          | List all configuration values (password redacted)          |
| `cz-otel config validate`      | Validate that all required fields are present and valid    |
| `cz-otel config generate`      | Generate collector YAML config from stored settings        |
| `cz-otel start`                | Start the collector as a background daemon                 |
| `cz-otel start --foreground`   | Run the collector in the foreground                        |
| `cz-otel stop`                 | Stop the running collector daemon                          |
| `cz-otel restart`              | Restart the collector with current configuration           |
| `cz-otel status`               | Check if the collector is running (PID, uptime)            |
| `cz-otel logs`                 | Show last 50 lines of collector logs                       |
| `cz-otel logs --follow`        | Stream collector logs in real time                         |
| `cz-otel version`              | Print CLI version, collector version, and platform         |
| `cz-otel --help`               | Show help for all commands                                 |

## Configuration

All configuration is stored in `~/.cz-otel/config.yaml` (macOS/Linux) or `%APPDATA%\cz-otel\config.yaml` (Windows).

| Key                  | Required | Default        | Description                              |
|----------------------|----------|----------------|------------------------------------------|
| `service`            | Yes      | —              | ClickZetta service URL                   |
| `username`           | Yes      | —              | ClickZetta username                      |
| `password`           | Yes      | —              | ClickZetta password                      |
| `workspace`          | Yes      | —              | Target workspace name                    |
| `virtual_cluster`    | Yes      | —              | Virtual cluster name                     |
| `instance`           | Yes      | —              | Instance name                            |
| `schema`             | No       | `public`       | Target schema                            |
| `protocol`           | No       | `https`        | Connection protocol                      |
| `logs_table_name`    | No       | `otel_logs`    | Table name for log data                  |
| `traces_table_name`  | No       | `otel_traces`  | Table name for trace data                |
| `metrics_table_name` | No       | `otel_metrics` | Table name for metrics data              |
| `create_schema`      | No       | `true`         | Auto-create schema if it doesn't exist   |

The config file is stored with `0600` permissions on Unix to protect credentials.

## Building from Source

### Prerequisites

- Go 1.22+
- OCB (OpenTelemetry Collector Builder) — install with:
  ```bash
  go install go.opentelemetry.io/collector/cmd/builder@latest
  ```

### Build Targets

```bash
# Build the CLI binary for the current platform
make build

# Build the collector binary using OCB
make collector

# Create a distribution package (tar.gz) for the current platform
make package

# Cross-compile and package for all supported platforms
make package-all

# Run tests
make test

# Remove build artifacts
make clean
```

Version can be overridden at build time:
```bash
make build VERSION=1.2.3
make package VERSION=1.2.3
```

## Platform Support

| Platform        | Architecture | Package Name                              |
|-----------------|--------------|-------------------------------------------|
| macOS           | amd64        | `cz-otel-<version>-darwin-amd64.tar.gz`   |
| macOS           | arm64 (M1+)  | `cz-otel-<version>-darwin-arm64.tar.gz`   |
| Linux           | amd64        | `cz-otel-<version>-linux-amd64.tar.gz`    |
| Linux           | arm64        | `cz-otel-<version>-linux-arm64.tar.gz`    |
| Windows         | amd64        | `cz-otel-<version>-windows-amd64.tar.gz`  |

## File Layout

After installation, the following directory structure is created:

```
~/.cz-otel/
├── bin/
│   ├── cz-otel                    # CLI binary
│   └── otelcol-clickzetta         # Collector binary
├── logs/
│   └── collector.log              # Collector log output
├── config.yaml                    # User configuration
├── otel-collector-config.yaml     # Generated collector config
└── collector.pid                  # PID file (when running)
```

On Windows, the base directory is `%APPDATA%\cz-otel\`.
