# ADR-018: Grafana LGTM Stack for Observability Backends

## Status

Accepted

## Context

ADR-006 established OpenTelemetry + slog as the instrumentation layer from day 1. The original plan specified Prometheus, Jaeger, Loki, and Grafana as four separate backend systems. This works but means deploying and maintaining four independent systems with different storage backends, APIs, and operational models.

Grafana has been converging its observability stack into a unified platform (LGTM: Loki, Grafana, Tempo, Mimir). All components are open-source (AGPL-3.0), self-hostable, and use S3-compatible object storage (MinIO) — which PatchIQ already has in the stack.

## Decision

Use the Grafana LGTM stack for observability backends:

| Component | Replaces | Purpose | Storage |
|-----------|----------|---------|---------|
| **Mimir** | Prometheus (long-term storage) | Metrics | S3/MinIO |
| **Tempo** | Jaeger | Distributed tracing | S3/MinIO |
| **Loki** | — (was already Loki) | Log aggregation | S3/MinIO |
| **Grafana** | — (was already Grafana) | Unified dashboards | — |

Note: Prometheus is still used as the scrape/collection layer. Mimir provides horizontally-scalable long-term storage with the same PromQL query language.

**Deployment strategy:**
- **Phase 1 / early SaaS development**: Use Grafana Cloud free tier for Hub Manager. Minimal ops overhead while focused on product development.
- **Scale / production**: Migrate to self-hosted LGTM on AWS (EC2/EKS) or own infrastructure. All data formats and APIs are identical — OTel instrumentation doesn't change.
- **On-prem customers**: Self-host LGTM alongside PatchIQ, or point OTel exporters at their existing monitoring stack.

## Consequences

- **Positive**: Single vendor for all observability backends; unified storage on S3/MinIO (already in stack); unified query experience in Grafana; self-hostable with no license restrictions on functionality; Grafana Cloud free tier accelerates early development; OTel instrumentation stays vendor-neutral
- **Negative**: AGPL-3.0 license (no issue for internal tooling, but worth noting); Mimir/Tempo are newer than Prometheus/Jaeger (less battle-tested at extreme scale); Grafana Labs controls the roadmap for all four components

## Alternatives Considered

- **Prometheus + Jaeger + Loki + Grafana (original plan)**: Four separate systems — rejected because more operational overhead; different storage backends; same end result with more complexity
- **Minimal (Prometheus + Grafana only)**: Start light, add tracing/logs later — rejected because tracing is essential for debugging distributed agent↔server↔hub communication from day 1
- **Datadog/New Relic**: Commercial APM — rejected because customers may be air-gapped; vendor lock-in; cost scales with agent count
