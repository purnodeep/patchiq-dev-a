# ADR-007: Hub-Spoke Topology with Distribution Servers

## Status

Accepted

## Context

Enterprise customers have branch offices with limited WAN bandwidth. Agents at remote sites shouldn't download patches over slow WAN links individually.

## Decision

Support hub-spoke topology with distribution servers (lightweight Patch Manager components) at branch offices that cache binaries locally and relay agent data to the parent.

## Consequences

- **Positive**: Mirrors proven WSUS/ManageEngine patterns; bandwidth-efficient; branch offices don't need full PM; distribution servers support up to 5,000 agents; offline resilience via local SQLite cache
- **Negative**: Additional component to build and maintain; parent-child sync logic adds complexity; eventual consistency between sites; distribution server monitoring needed

## Alternatives Considered

- **Single centralized server**: All agents connect to one PM — rejected because doesn't scale for remote sites with slow WAN
- **Full PM at every site**: Deploy complete Patch Manager at branches — rejected because operational overhead and cost; branches don't need full autonomy
- **CDN-based distribution**: Use CDN for patch binary delivery — rejected because doesn't work for air-gapped/internal networks
