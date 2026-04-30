# Observability Configs

This directory holds reference configuration for observing Dramora in production.
Files here are not loaded by the application itself; they're meant to be copied or
referenced from your Prometheus / Alertmanager / Grafana deployment.

## Files

| File | Purpose |
| ---- | ------- |
| `prometheus-worker-metrics.rules.yaml` | Sample Prometheus alert + recording rules for the worker org-context metrics exposed at `/metrics`. |

## Worker metrics

The Dramora API exposes worker observability counters as Prometheus 0.0.4 text
exposition at `GET /metrics` (root, public). Currently exposed:

- `dramora_worker_org_unresolved_skips_total{kind="generation"|"export"}` — counter
  incremented when the worker skipped a queued job because it could not resolve
  the owning organization context.
- `dramora_worker_last_skip_timestamp_seconds` — gauge (Unix seconds) of the
  most recent skip; `0` if none has happened.
- `dramora_worker_last_skip_info{kind, reason}` — gauge (always `1`) carrying the
  most recent skip's labels for diagnostic dashboards.

The same data is mirrored at `GET /api/v1/admin/worker-metrics` for the Studio
admin UI (owner / admin role required); when a persistent metrics store is
configured the JSON snapshot's `source` field reports `aggregated` to indicate
cross-process totals.

## Suggested scrape config

```yaml
scrape_configs:
  - job_name: dramora-worker
    metrics_path: /metrics
    static_configs:
      - targets: ['api.example.com']
```

`/metrics` is intentionally unauthenticated; restrict access at the network or
reverse-proxy layer.
