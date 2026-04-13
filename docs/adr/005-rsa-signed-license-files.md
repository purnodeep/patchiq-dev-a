# ADR-005: RSA-Signed JSON License Files

## Status

Accepted

## Context

PatchIQ must support air-gapped and offline deployments. License validation cannot depend on internet connectivity.

## Decision

Use RSA-2048 signed JSON license files with the public key embedded in the Patch Manager binary at build time.

## Consequences

- **Positive**: Works offline/air-gapped; tamper-proof (cryptographic signature); feature gating is trivial (read JSON fields); grace period logic is simple
- **Negative**: License rotation requires binary update or key rotation strategy; no real-time revocation (grace period mitigates); clock manipulation could extend expiry (mitigated by 48h drift tolerance)

## Alternatives Considered

- **Online license server**: Real-time validation — rejected because breaks air-gapped requirement
- **Hardware dongles**: Physical license — rejected because doesn't scale to cloud/container deployments
- **Time-limited tokens with refresh**: OAuth-style — rejected because requires connectivity
