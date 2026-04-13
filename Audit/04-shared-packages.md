# Shared Packages Audit (`internal/shared/`)

**Audited by:** Claude Opus 4.6
**Date:** 2026-04-09
**Branch:** dev-a (commit 7b426cb)
**Test status:** All 9 packages pass (`go test -race -short`) -- 0 failures

---

## Executive Summary

The shared packages are well-structured, consistent, and have good test coverage. No import violations found. The main issues are: (1) duplicate UserID context keys across `user/` and `otel/`, (2) several exports unused by any platform, (3) stale milestone comments, and (4) minor crypto considerations. No critical security issues.

---

## Import Violations

**None found.** Verified that `internal/shared/` has zero imports from `internal/server/`, `internal/hub/`, or `internal/agent/`. The only cross-shared-package import is `otel/slog.go` importing `shared/tenant` (valid).

---

## Package-by-Package Findings

### 1. `tenant/` -- Context + Middleware

**Files:** `context.go`, `middleware.go`, `context_test.go`, `middleware_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| T-1 | Minor | Stale comment: "M0 stub: in M2+ this will also support JWT claims, API keys, and subdomain extraction." M2 is marked complete per CLAUDE.md but the comment remains and JWT/API-key extraction was implemented in `server/auth/`, not here. | `middleware.go:17` |
| T-2 | Minor | `WithTenantID` panics on empty string (line 10). This is intentional defensive coding but means a bug in caller code causes a crash instead of a graceful error. Consistent with `user/` package so not a defect, but worth noting. | `context.go:10` |

**Test coverage:** Good. Tests cover round-trip, missing context, panic cases, valid/missing/invalid UUID middleware paths. 7 test cases total.

**Security:** UUID validation is correct -- rejects non-UUID strings. Middleware correctly returns 400 before passing through.

---

### 2. `user/` -- Context Only (No Middleware)

**Files:** `context.go`, `context_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| U-1 | Minor | No UUID validation on user IDs. `WithUserID` accepts any non-empty string (e.g., `"user-123"`) unlike `tenant/middleware.go` which validates UUIDs. This is by design since user IDs come from Zitadel (which uses its own format), but worth documenting. | `context.go:8-12` |

**Test coverage:** Good. 4 test cases covering round-trip, empty-context, panic-on-empty, MustUserID.

---

### 3. `domain/` -- Events + Audit + Bus Interface

**Files:** `events.go`, `audit.go`, `bus.go`, `events_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| D-1 | Minor | `ActorAIAssistant` constant ("ai_assistant") is exported but never used by any platform code. It maps to a DB CHECK constraint column so it should remain, but is dead code today. | `events.go:15` |
| D-2 | Minor | `NewAuditEvent` takes 9 positional parameters. This is error-prone and hard to read at call sites. An options struct or builder pattern would improve clarity. Not urgent since the API is stable. | `audit.go:6-16` |

**Test coverage:** Good. Tests cover ULID generation (length, uniqueness, ordering), JSON round-trip, NewAuditEvent, NewSystemEvent. 5 test functions.

---

### 4. `crypto/` -- AES-256-GCM + RSA

**Files:** `aes.go`, `rsa.go`, `aes_test.go`, `rsa_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| C-1 | Important | RSA-2048 key size. NIST recommends RSA-3072+ for use beyond 2030. Since PatchIQ is enterprise software with multi-year deployments, 2048-bit keys may be insufficient for long-lived license signing keys. Consider RSA-4096 or Ed25519. | `rsa.go:14` (hardcoded `2048`) |
| C-2 | Minor | `SignPayload` uses `PKCS1v15` signing instead of PSS. PKCS1v15 is not broken but PSS is the modern recommendation with provable security. This is a low risk since it's only used for license signing (not TLS). | `rsa.go:25` |
| C-3 | Minor | `GenerateKey()` panics on `rand.Reader` failure (line 18). While this is extremely unlikely (only if the OS entropy source fails), a returned error would be safer for production use. | `aes.go:17-19` |
| C-4 | Minor | `DecodePrivateKeyPEM` discards the rest-of-PEM data (`block, _ := pem.Decode(data)`) on line 51. If a PEM file contains multiple blocks, only the first is used silently. This is standard practice but worth noting. | `rsa.go:51` |

**Test coverage:** Excellent. AES tests cover round-trip, wrong-key rejection, nonce uniqueness. RSA tests cover key generation, sign/verify, tampered payloads, truncated sigs, wrong keys, PEM round-trips, invalid PEM. 6 test functions with multiple subtests.

**Security:** AES-256-GCM implementation is correct -- uses `crypto/rand` for nonce, prepends nonce to ciphertext. No hardcoded keys or IVs. The `Decrypt` function correctly validates ciphertext length before slicing.

---

### 5. `idempotency/` -- HTTP Middleware + Command Deduplication

**Files:** `middleware.go`, `cache.go`, `command.go`, `middleware_test.go`, `cache_test.go`, `command_test.go`, `optimistic_test.go`, `cache_integration_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| I-1 | Important | No in-flight request deduplication. Two concurrent requests with the same idempotency key can both execute. The comment acknowledges this: "In-flight locking can be added in M1 if needed." M1 is complete but this was not implemented. For enterprise deployments with HA/load balancing, this is a real risk. | `middleware.go:24` |
| I-2 | Minor | `MemoryStore` does not enforce TTL (line 124: "TTL is accepted but not enforced"). This is fine for tests but the hub uses `MemoryStore` in production (`cmd/hub/main.go:301`). Long-running hub instances will accumulate stale idempotency entries indefinitely. | `cache.go:124` |
| I-3 | Minor | `CommandStore` and `Deduplicator` types are exported but have zero usage in server/hub/agent production code. Only used in their own tests. These were designed for the agent's gRPC command deduplication but the agent doesn't import them. | `command.go:18-32` |

**Test coverage:** Excellent. 14+ test functions covering middleware (all HTTP methods, cache hit/miss, error stores, no-tenant), memory store (get/set, miss, tenant isolation, different keys), command deduplicator (new, duplicate, different, handler error, store errors), optimistic locking, and Valkey integration tests (behind build tag). Best-tested package in shared/.

---

### 6. `config/` -- Loader + Hierarchy + Types + Defaults

**Files:** `loader.go`, `types.go`, `hierarchy.go`, `sources.go`, `defaults.go`, `store.go`, `logwriter.go`, `loader_test.go`, `hierarchy_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| CF-1 | Important | Config hierarchy system (`MergeConfig`, `ResolveConfig`, `ConfigStore`, `ResolveParams`, `ResolvedConfig`, all Default*Config functions, all module/scope constants, all config types like `ScanConfig`, `DeployConfig`, `NotificationConfig`, `AgentConfig`, `CVEConfig`) is entirely unused by any platform code. Only `DiscoveryConfig` and `RepositoryConfig` are used (by `server/discovery/`). This is a large amount of dead code. | `hierarchy.go`, `sources.go`, `defaults.go`, `store.go`, `types.go` |
| CF-2 | Minor | `knownKeys` and `hubKnownKeys` maps are manually maintained and must be kept in sync with struct field names. If a new snake_case field is added to a config struct, the env var override will silently fail unless the map is updated. No validation exists to catch drift. | `loader.go:110-134`, `loader.go:201-226` |
| CF-3 | Minor | `ServerConfig.validate()` does not validate HTTP timeout fields (read_timeout, write_timeout, idle_timeout, shutdown_timeout) unlike `HubConfig.validate()` which does. A zero-value server read_timeout would be accepted silently. | `loader.go:288-302` |

**Test coverage:** Good. Loader tests cover file loading, env overrides, missing file, validation failures (port bounds, missing DB URL, same ports). Hierarchy tests thoroughly cover merge semantics (nil preservation, scalar override, slice dedup, struct pointer replacement) and 4-level config resolution. 2 test files with extensive subtests.

---

### 7. `otel/` -- OpenTelemetry Init + Middleware + Slog Handler + Context

**Files:** `init.go`, `middleware.go`, `slog.go`, `context.go`, `grpc.go`, `init_test.go`, `middleware_test.go`, `slog_test.go`, `context_test.go`, `grpc_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| O-1 | Important | Duplicate UserID context key. `otel/context.go` defines `WithUserID`/`UserIDFromContext` using `userIDKey{}`, while `user/context.go` defines the same functions using `userCtxKey{}`. These are different context keys, meaning code that sets the user ID via `user.WithUserID` cannot be read by `otel.UserIDFromContext` (used in the slog handler). Currently `otel.WithUserID` is never called from app code, so user IDs are **never injected into structured logs** despite the slog handler supporting it. | `otel/context.go:18-29` vs `user/context.go:5-12` |
| O-2 | Minor | `GRPCClientHandler()` is exported but never used by any platform. The agent uses direct gRPC dial without the OTel handler. | `grpc.go:14-16` |

**Test coverage:** Excellent. Tests cover noop init (empty endpoint), real tracer init, HTTP middleware span creation, slog handler (trace_id, tenant_id, request_id, user_id injection, empty field omission, WithAttrs/WithGroup preservation), gRPC handler nil checks, context round-trips. 13 test functions.

---

### 8. `license/` -- Tiers + Types

**Files:** `tiers.go`, `types.go`, `tiers_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| L-1 | Minor | `LicenseStatus` and `EndpointUsage` types are defined here but appear to be response-specific DTOs. They are not used in server or hub code (the server builds its own response structs). May be dead code. | `types.go:57-74` |
| L-2 | Minor | CLAUDE.md documents tiers as "FREE, STANDARD, ENTERPRISE" but actual constants are `community`, `professional`, `enterprise`, `msp`. Documentation is out of sync. | `types.go:49-54` |

**Test coverage:** Good. Tests cover all 4 tiers (endpoint limits, SSO flags), invalid tier error, FeatureMap for enterprise and community. 4 test functions.

---

### 9. `protocol/` -- Version Negotiation

**Files:** `version.go`, `version_test.go`

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| P-1 | -- | No issues found. Clean, minimal, well-tested. | -- |

**Test coverage:** Good. 6 test cases covering same version, agent newer, server newer, agent below min, zero agent, zero server.

---

### 10. `models/` -- Missing Package

| # | Severity | Finding | Location |
|---|----------|---------|----------|
| M-1 | Minor | CLAUDE.md lists `models/` as a shared package ("Shared domain types") but the directory is empty (no .go files). Either the package was never created or its contents were moved elsewhere. Documentation should be updated. | `internal/shared/models/` (empty) |

---

## Cross-Cutting Findings

| # | Severity | Finding | Details |
|---|----------|---------|---------|
| X-1 | Important | **Duplicate UserID context** between `user/` and `otel/` packages. The slog handler reads `otel.UserIDFromContext()` but app code sets `user.WithUserID()`. These use different context keys, so user IDs never appear in structured logs. Fix: have the otel slog handler read from `user.UserIDFromContext()` instead of its own context key, or have the auth middleware set both. | `otel/context.go:18-29`, `otel/slog.go:66-68`, `user/context.go` |
| X-2 | Minor | **Stale milestone references.** "M0 stub" in `tenant/middleware.go:17`, "M1 if needed" in `idempotency/middleware.go:24`. Both milestones are complete. Comments should be updated to reflect current state or replaced with issue references per anti-slop rule #10. | |
| X-3 | Minor | **Consistent panic pattern.** `tenant.WithTenantID`, `user.WithUserID`, and `crypto.GenerateKey` all panic on invalid input. This is consistent across packages (good) but means caller bugs cause crashes rather than graceful errors. Acceptable for invariant violations but should be documented. | |
| X-4 | Minor | **No `models/` package.** CLAUDE.md references it but it does not exist. | |

---

## Dead/Unused Export Summary

| Export | Package | Used In Tests Only | Used In Production |
|--------|---------|-------------------|--------------------|
| `ActorAIAssistant` | domain | No | No |
| `GRPCClientHandler` | otel | Test only | No |
| `otel.WithUserID` / `otel.UserIDFromContext` | otel | Test only | No (app uses `user.WithUserID`) |
| `CommandStore` / `Deduplicator` / `CommandResult` / `MemoryCommandStore` | idempotency | Test only | No |
| `MergeConfig` / `ResolveConfig` / `ConfigStore` / `ResolveParams` / `ResolvedConfig` | config | Test only | No |
| `DefaultScanConfig` / `DefaultDeployConfig` / `DefaultNotificationConfig` / `DefaultCVEConfig` / `DefaultAgentConfig` | config | Test only | No |
| `ScanConfig` / `DeployConfig` / `NotificationConfig` / `AgentConfig` / `CVEConfig` | config | Test only | No |
| `Module*` / `Scope*` constants | config | Test only | No |
| `SourceLevel` / `SourceSystem` / `SourceTenant` / `SourceGroup` / `SourceEndpoint` | config | Test only | No |
| `LicenseStatus` / `EndpointUsage` | license | No | No |
| `FeatureMap` | license | Test only | No |

**Note:** Many of these (config hierarchy, command deduplication) are designed infrastructure waiting for feature adoption. They are not accidental dead code -- they were built ahead of usage. However, per anti-slop rule #6 ("do exactly what was asked"), this represents speculative code.

---

## Severity Summary

| Severity | Count | Key Items |
|----------|-------|-----------|
| Critical | 0 | -- |
| Important | 4 | Duplicate UserID context (X-1/O-1), no in-flight dedup (I-1), RSA-2048 (C-1), large dead config hierarchy (CF-1) |
| Minor | 14 | Stale comments, dead exports, validation gaps, doc drift |

---

## Recommended Actions

1. **Fix UserID context duplication** (Important): Have `otel/slog.go` import and use `user.UserIDFromContext` instead of its own duplicate. Remove `otel.WithUserID`/`otel.UserIDFromContext` or keep only for slog injection.
2. **Add in-flight idempotency locking** (Important): Implement Redis/Valkey-based locking for concurrent request deduplication, or accept the risk and update the comment from "M1" to a specific issue number.
3. **Evaluate RSA key size** (Important): Upgrade `GenerateKeyPair` to RSA-4096 or Ed25519 for new license signing keys. Existing signed licenses would need re-signing.
4. **Audit config hierarchy usage** (Important): Either wire the config hierarchy into the server's settings API or remove it to reduce maintenance burden.
5. **Clean up stale milestone comments** (Minor): Replace "M0 stub" and "M1 if needed" with issue references.
6. **Add server config validation parity** (Minor): Add timeout validation to `ServerConfig.validate()` matching `HubConfig.validate()`.
7. **Fix CLAUDE.md** (Minor): Update license tier names, remove `models/` from package list.
8. **Hub MemoryStore TTL** (Minor): Add TTL enforcement to `MemoryStore` or switch hub to `ValkeyStore`.
