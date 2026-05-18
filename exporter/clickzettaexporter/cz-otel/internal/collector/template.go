// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package collector

// collectorConfigTemplate is the Go text/template for the OTel Collector configuration YAML.
// It uses placeholders for all clickzetta exporter values populated from the config store.
const collectorConfigTemplate = `receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1m
    send_batch_size: 8192
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128

exporters:
  clickzetta:
    service: {{.service}}
    username: {{.username}}
    password: {{.password}}
    workspace: {{.workspace}}
    virtual_cluster: {{.virtual_cluster}}
    instance: {{.instance}}
    schema: {{.schema}}
    create_schema: {{.create_schema}}
    logs_table_name: {{.logs_table_name}}
    traces_table_name: {{.traces_table_name}}
    metrics_table_name: {{.metrics_table_name}}

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickzetta]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickzetta]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [clickzetta]
`
