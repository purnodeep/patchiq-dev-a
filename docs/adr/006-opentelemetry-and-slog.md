# ADR-006: OpenTelemetry + slog from Day 1

## Status

Accepted (Updated — observability backends moved to Grafana LGTM per ADR-018)

## Context

PatchIQ is a distributed system (agents, server, hub). Debugging production issues without observability is guesswork. Retrofitting instrumentation later means incomplete coverage.

## Decision

Instrument all three platforms with OpenTelemetry from day 1. Use Go stdlib `slog` for structured logging with JSON handler. Observability backends use the Grafana LGTM stack (see ADR-018: Mimir for metrics, Tempo for tracing, Loki for logs, Grafana for dashboards).

## Consequences

- **Positive**: Essential for debugging at scale; enables support bundle generation; AI-friendly structured logs; correlation IDs across services; zero external logging dependency (slog is stdlib); OTel instrumentation is vendor-neutral — backends can be swapped without code changes
- **Negative**: OTel SDK adds some overhead; team must learn OTel conventions; `sloglint` in CI adds friction; Grafana LGTM stack has operational cost (mitigated by Grafana Cloud free tier for early development)

## Alternatives Considered

- **zerolog**: Faster but non-stdlib — rejected because slog is now stdlib and good enough; `sloglint` enforcement is easier
- **Add observability later**: Defer to Phase 2 — rejected because the data must exist before you can debug with it; retrofitting is painful
- **Datadog/New Relic SaaS**: Commercial APM — rejected because customers may be air-gapped; vendor lock-in; cost
