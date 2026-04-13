# ADR-011: Valkey over Redis

## Status

Accepted

## Context

PatchIQ is a commercial product with both SaaS (Hub Manager) and on-prem (Patch Manager) deployment models. Redis changed its license to SSPL/RSALv2 in March 2024 (later adding AGPLv3 in May 2025). SSPL requires releasing the entire service stack source code if offering Redis as a hosted service. Even AGPLv3 has copyleft implications for SaaS. We need a cache/KV store with clear commercial licensing.

Valkey is a Linux Foundation fork of Redis (forked from Redis OSS 7.2), licensed under BSD-3. It is 100% API-compatible and backed by AWS, Google, Oracle, Alibaba, Ericsson, and others.

## Decision

Use Valkey 9.0 as the cache/KV store across all PatchIQ platforms. All existing Redis client libraries (go-redis, etc.) work unchanged.

## Consequences

- **Positive**: BSD-3 license — no commercial restrictions for SaaS or on-prem; 40% better throughput in Valkey 9.0; backed by Linux Foundation with major cloud vendors; all managed services (AWS ElastiCache, Google Memorystore) now offer Valkey; drop-in replacement — zero code changes
- **Negative**: Smaller independent community (growing fast); some Redis-specific newer features (Redis 8+) may diverge; documentation still references "Redis" in many places

## Alternatives Considered

- **Redis (SSPL/AGPLv3)**: Feature-original but license risk for commercial SaaS — rejected due to licensing implications for Hub Manager
- **DragonflyDB**: High-performance Redis-compatible — rejected because custom BSL license; single-company risk
- **KeyDB**: Multi-threaded Redis fork — rejected because acquired by Snap, uncertain future
