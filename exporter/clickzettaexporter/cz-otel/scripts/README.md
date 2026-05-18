# cz-otel - ClickZetta OpenTelemetry Collector CLI

A command-line tool for configuring and running the OpenTelemetry Collector with the ClickZetta exporter plugin. Sends traces, metrics, and logs to ClickZetta Lakehouse with minimal setup.

## Quick Start

1. **Run the install script**

   On macOS/Linux:
   ```bash
   ./install.sh
   ```

   On Windows:
   ```cmd
   install.bat
   ```

2. **Add to PATH** (if needed)

   The install script will print instructions if `~/.cz-otel/bin` is not already in your PATH.

3. **Configure your connection**

   ```bash
   cz-otel config init
   ```

   This interactive wizard prompts for your ClickZetta service URL, credentials, workspace, and other settings.

4. **Start the collector**

   ```bash
   cz-otel start
   ```

   The collector runs as a background daemon, receiving OTLP data on ports 4317 (gRPC) and 4318 (HTTP).

## Available Commands

| Command              | Description                                              |
|----------------------|----------------------------------------------------------|
| `config init`        | Interactive setup wizard for all configuration fields    |
| `config set <k> <v>` | Set a single configuration value                        |
| `config get <key>`   | Get a single configuration value                        |
| `config list`        | List all configuration values (password redacted)        |
| `config validate`    | Validate that all required fields are configured         |
| `config generate`    | Generate the collector YAML config from stored settings  |
| `start`             | Start the collector as a background daemon               |
| `stop`              | Stop the running collector daemon                        |
| `restart`           | Restart the collector with current configuration         |
| `status`            | Check if the collector is running                        |
| `logs`              | View collector log output                                |
| `version`           | Print version information                                |

## Configuration

The following configuration keys are supported:

| Key                | Required | Default        | Description                          |
|--------------------|----------|----------------|--------------------------------------|
| `service`          | Yes      | —              | ClickZetta service URL               |
| `username`         | Yes      | —              | ClickZetta username                  |
| `password`         | Yes      | —              | ClickZetta password                  |
| `workspace`        | Yes      | —              | Target workspace name                |
| `virtual_cluster`  | Yes      | —              | Virtual cluster name                 |
| `instance`         | Yes      | —              | Instance name                        |
| `schema`           | No       | `public`       | Target schema                        |
| `protocol`         | No       | `https`        | Connection protocol                  |
| `logs_table_name`  | No       | `otel_logs`    | Table name for log data              |
| `traces_table_name`| No       | `otel_traces`  | Table name for trace data            |
| `metrics_table_name`| No      | `otel_metrics` | Table name for metrics data          |
| `create_schema`    | No       | `true`         | Auto-create schema if it doesn't exist |

Configuration is stored in `~/.cz-otel/config.yaml` (macOS/Linux) or `%APPDATA%\cz-otel\config.yaml` (Windows).

## Troubleshooting

### Collector fails to start

- Run `cz-otel config validate` to check for missing required fields.
- Ensure the collector binary exists at `~/.cz-otel/bin/otelcol-clickzetta`. If missing, re-run the install script.

### Connection errors

- Verify your `service` URL is correct and reachable.
- Check that `username` and `password` are valid.
- Ensure `workspace`, `virtual_cluster`, and `instance` match your ClickZetta environment.

### Port conflicts

The collector listens on ports 4317 (gRPC) and 4318 (HTTP) by default. If these ports are in use, stop the conflicting process or configure your applications to use different ports.

### Viewing logs

```bash
cz-otel logs            # Show last 50 lines
cz-otel logs --follow   # Stream logs in real time
```

Log files are stored at `~/.cz-otel/logs/collector.log` (macOS/Linux) or `%APPDATA%\cz-otel\logs\collector.log` (Windows).

### Stale PID file

If `cz-otel status` reports the collector as running but it isn't responding, the PID file may be stale. Run `cz-otel stop` to clean it up, then `cz-otel start` to restart.
