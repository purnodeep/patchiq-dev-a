# Feature: Observability & APM

> Status: In Progress | Phase: 1 (instrumented from day 1) | License: All tiers

---

## Overview

All three platforms (Agent, Patch Manager, Hub Manager) are instrumented with OpenTelemetry from day 1 for metrics, traces, and structured logs. This enables debugging, support bundle generation, and AI-friendly log analysis.

## Problem Statement

Patch management at scale involves distributed systems (thousands of agents, server components, background jobs). Without observability from the start, debugging production issues becomes guesswork. Retrofitting instrumentation later means incomplete coverage during the critical early deployment phase.

---

## Observability Stack

```
┌──────────┐  ┌──────────────┐  ┌──────────────┐
│  Agent   │  │ Patch Manager│  │ Hub Manager  │
│          │  │              │  │              │
│ OTel SDK │  │  OTel SDK    │  │  OTel SDK    │
└────┬─────┘  └──────┬───────┘  └──────┬───────┘
     │               │                 │
     └───────────────┼─────────────────┘
                     │ OTLP (gRPC)
              ┌──────┴───────┐
              │ OTel Collector│
              └──────┬───────┘
                     │
         ┌───────────┼───────────┐
         │           │           │
    ┌────┴───┐  ┌───┴────┐  ┌──┴─────┐
    │Prometheus│ │ Jaeger │  │  Loki  │
    │(Metrics) │ │(Traces)│  │ (Logs) │
    └────┬───┘  └───┬────┘  └──┬─────┘
         │          │           │
         └──────────┼───────────┘
                    │
              ┌─────┴─────┐
              │  Grafana   │
              │ Dashboards │
              └────────────┘
```

---

## What We Track

**Agent Metrics:**
- Heartbeat latency (agent → server round-trip)
- Inventory scan duration
- Patch installation duration and exit codes
- Agent memory/CPU usage
- gRPC connection state and reconnection count
- Offline queue depth

**Patch Manager Metrics:**
- API response times (p50, p95, p99)
- Active WebSocket connections
- Database query latency
- Redis queue depth and processing rate
- Job execution rate (patches/minute)
- Active deployment count
- Authentication success/failure rates

**Hub Manager Metrics:**
- Patch feed sync duration and freshness
- License validation request rate
- Catalog size and update frequency
- Client connection count and health

---

## Structured Logging Strategy

**Library**: Go `slog` (stdlib, zero external dependency) with JSON handler

**Standards enforced from day 1:**
- Every log line includes: `trace_id`, `span_id`, `request_id`, `user_id` (if authenticated), `tenant_id` (if multi-tenant)
- Correlation IDs propagated via Go `context.Context` across all service boundaries
- Log levels: DEBUG (development), INFO (operations), WARN (degraded), ERROR (failures)
- Sensitive data (passwords, tokens, PII) never logged — enforced via `LogValuer` interface on sensitive types
- `sloglint` integrated in CI to enforce structured logging patterns

**Why this matters for debugging:**
Every log entry has enough context for an AI assistant or human operator to:
1. Trace a request across services using `trace_id`
2. Identify which user/tenant triggered the action
3. Correlate errors with the specific endpoint/deployment that failed
4. Reconstruct the sequence of events without needing a debugger

---

## Phone-Home Telemetry (Opt-In)

For product improvement, Patch Manager instances can optionally send anonymized telemetry to the Hub:

**What we collect (if opted in):**
- Feature usage counts (which features are used most)
- Endpoint count and OS distribution
- Patch deployment success/failure rates
- Error categories (not messages — just `PATCH_INSTALL_FAILED`, not stack traces)
- API endpoint usage patterns
- Performance percentiles

**What we never collect:**
- Hostnames, IPs, or any endpoint-identifying information
- Patch names or CVE IDs (only counts)
- User names, emails, or credentials
- Any customer data or configuration details

**Controls:**
- Off by default, enabled in settings with clear disclosure
- Per-category toggles (usage, performance, errors)
- Data review: admins can preview exactly what will be sent before enabling
- Standard: GDPR/CCPA compliant, data retention max 90 days

---

## Integration Points

| Feature | Integration |
|---------|-------------|
| **Support System** | Support bundle generator pulls from OTel data |
| **AI Assistant** | AI can query logs/metrics to diagnose issues |
| **Health Dashboard** | `/health` endpoint reports component status |

---

## Code Mapping

| Area | Code Directory |
|------|---------------|
| OTel initialization (shared) | `internal/common/otel/` |
| Server APM | `internal/server/apm/` |
| Agent APM | `internal/agent/apm/` |
