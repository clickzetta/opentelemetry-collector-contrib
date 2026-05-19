# API Key Routing Connector

| Status        |           |
| ------------- |-----------|
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aconnector%2Fapikeyrouting%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aconnector%2Fapikeyrouting) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aconnector%2Fapikeyrouting%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aconnector%2Fapikeyrouting) |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib

## Supported Pipeline Types

| [Exporter Pipeline Type] | [Receiver Pipeline Type] | [Stability Level] |
| ------------------------ | ------------------------ | ----------------- |
| traces | traces | [alpha] |
| metrics | metrics | [alpha] |
| logs | logs | [alpha] |

[Exporter Pipeline Type]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#exporter-pipeline-type
[Receiver Pipeline Type]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md#receiver-pipeline-type
[Stability Level]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#stability-levels

## Overview

The API Key Routing Connector dynamically routes telemetry (logs, traces, metrics) to downstream pipelines based on an API key extracted from request metadata. It resolves API keys against an external Key Service to determine routing targets and can dynamically create exporter instances at runtime based on the Key Service response.

This connector is **exporter-agnostic** — it routes to pipeline IDs and dynamically creates exporters of any registered type, allowing centralized routing management without collector redeployment.

## Configuration

If you are not already familiar with connectors, you may find it helpful to first visit the [Connectors README].

[Connectors README]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/connector/README.md

### Settings

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `key_service_url` | string | *(required)* | Base URL of the Key Service API. The connector appends `/v1/api/keys/{key}` to this URL. |
| `key_header` | string | `x-api-key` | The Client_Metadata key to extract the API key from. |
| `cache_ttl` | duration | `5m` | Duration to cache Key Service responses. Must be positive. |
| `key_service_timeout` | duration | `5s` | HTTP request timeout for Key Service calls. Must be positive. |
| `default_pipelines` | []pipeline.ID | *(empty)* | Pipeline IDs to route to when a key cannot be resolved or is missing. If empty, unresolved telemetry is dropped with a warning. |
| `default_exporter_type` | string | `clickzetta` | Exporter type to use when the Key Service response does not include an `exporter_type` field. |
| `exporter_idle_timeout` | duration | `30m` | Duration after which an unused dynamic exporter is shut down and evicted. |

### Example

```yaml
connectors:
  apikeyrouting:
    key_service_url: "http://key-service:8080"
    key_header: "x-api-key"
    cache_ttl: 5m
    key_service_timeout: 5s
    default_exporter_type: "clickzetta"
    exporter_idle_timeout: 30m
    default_pipelines:
      - logs/default
```

## Key Service Response Schema

The connector resolves API keys by calling:

```
GET {key_service_url}/v1/api/keys/{api_key}
```

### Single Pipeline Response

When the Key Service resolves a key to a single pipeline target:

```json
{
  "pipeline_id": "tenant-a",
  "exporter_type": "clickzetta",
  "exporter_config": {
    "service": "lakehouse.example.com",
    "username": "otel_writer",
    "password": "secret",
    "workspace": "acme_workspace",
    "instance": "prod-01",
    "virtual_cluster": "vc_ingest",
    "schema": "observability"
  }
}
```

### Multi-Pipeline Response

When the Key Service resolves a key to multiple pipeline targets, include a `pipelines` array. When present and non-empty, the `pipelines` array takes precedence over the top-level fields:

```json
{
  "pipeline_id": "tenant-b",
  "exporter_type": "clickzetta",
  "exporter_config": {},
  "pipelines": [
    {
      "pipeline_id": "tenant-b-primary",
      "exporter_type": "clickzetta",
      "exporter_config": {
        "service": "primary.example.com",
        "username": "writer",
        "password": "secret",
        "workspace": "workspace_b",
        "instance": "prod-01",
        "virtual_cluster": "vc_ingest",
        "schema": "observability"
      }
    },
    {
      "pipeline_id": "tenant-b-analytics",
      "exporter_type": "otlp",
      "exporter_config": {
        "endpoint": "analytics.example.com:4317"
      }
    }
  ]
}
```

### Response Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pipeline_id` | string | yes | Logical pipeline identifier for this key. |
| `exporter_type` | string | yes* | Type of exporter to create (e.g., `clickzetta`, `otlp`, `file`). Falls back to `default_exporter_type` if omitted. |
| `exporter_config` | object | yes | Exporter-specific configuration passed to the exporter factory. |
| `pipelines` | array | no | List of pipeline targets for multi-pipeline routing. Takes precedence over top-level fields when present. |

Each entry in `pipelines` has the same structure: `pipeline_id`, `exporter_type`, and `exporter_config`.

Unknown fields in the response are ignored without error.

### Error Responses

| HTTP Status | Meaning | Connector Behavior |
|-------------|---------|-------------------|
| 200 | Key resolved successfully | Route to resolved pipeline, cache result |
| 401 | Key is invalid/unauthorized | Route to default pipeline, do not cache |
| 404 | Key not found | Route to default pipeline, do not cache |
| 5xx | Service unavailable | Use stale cache if available, else route to default |

## Important: Receiver Metadata Configuration

**Receivers must be configured with `include_metadata: true`** for the connector to access request headers (including the API key).

Without this setting, the API key header will not be available in the Client_Metadata context, and all telemetry will be routed to the default pipeline.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
    include_metadata: true  # Required for API key routing
```

## Processors in Upstream Pipeline

Standard processors such as `batch` and `memory_limiter` can be used in the upstream pipeline between receivers and the connector. Client_Metadata (including the API key header) is preserved through the processor chain.

```yaml
service:
  pipelines:
    logs/ingress:
      receivers: [otlp]
      processors: [memory_limiter, batch]  # Processors work fine here
      exporters: [apikeyrouting]
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Missing API key header | Route to default pipeline, log warning |
| Empty API key (whitespace only) | Route to default pipeline, log warning |
| Key Service returns 401/404 | Route to default pipeline, do not cache |
| Key Service returns 5xx | Use stale cache if available, else route to default pipeline |
| Key Service timeout | Use stale cache if available, else route to default pipeline |
| Key Service response not valid JSON | Treat as 5xx (stale cache fallback) |
| Dynamic exporter creation fails | Route to default pipeline, log error |
| Downstream consumer error | Propagate error upstream for retry handling |
| No default pipeline configured + unresolvable key | Drop telemetry, log error |

## Complete Example

See [example/otel-collector-config.yaml](./example/otel-collector-config.yaml) for a complete pipeline configuration.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
    include_metadata: true

processors:
  batch:
    send_batch_size: 1000
    timeout: 5s
  memory_limiter:
    check_interval: 1s
    limit_mib: 512

connectors:
  apikeyrouting:
    key_service_url: "http://key-service:8080"
    key_header: "x-api-key"
    cache_ttl: 5m
    key_service_timeout: 5s
    default_exporter_type: "clickzetta"
    exporter_idle_timeout: 30m
    default_pipelines:
      - logs/default

exporters:
  clickzetta/default:
    service: "default-lakehouse.example.com"
    username: "default_writer"
    password: "default_secret"
    workspace: "default_workspace"
    instance: "default-01"
    virtual_cluster: "vc_default"
    schema: "observability"

service:
  pipelines:
    # Upstream pipeline: receiver → processors → connector
    logs/ingress:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [apikeyrouting]
    # Default fallback pipeline (static)
    logs/default:
      receivers: [apikeyrouting]
      exporters: [clickzetta/default]
    # Tenant-specific pipelines are created dynamically by the connector
    # based on Key Service responses — no need to pre-configure them.
```
