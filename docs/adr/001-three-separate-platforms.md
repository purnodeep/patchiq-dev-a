# ADR-001: Three Separate Platforms

## Status

Accepted

## Context

PatchIQ needs to serve three different deployment contexts: endpoints (agents), client-site management (Patch Manager), and central operations (Hub Manager). We needed to decide whether to build a monolithic application or separate platforms.

## Decision

Build three distinct but interconnected platforms: Agent, Patch Manager, and Hub Manager. Each has its own binary, deployment model, and frontend.

## Consequences

- **Positive**: Clear separation of concerns; different deployment targets (endpoint vs. on-prem vs. cloud); Hub can evolve independently; agents stay lightweight
- **Negative**: More complex build/release pipeline; need to manage inter-platform API contracts; shared code must be carefully factored into `internal/common/`

## Alternatives Considered

- **Monolith**: Single binary for everything — rejected because agents need to be < 30MB and run on constrained endpoints
- **Two platforms (Agent + Server)**: Hub functionality embedded in Patch Manager — rejected because MSPs need a separate multi-tenant layer
