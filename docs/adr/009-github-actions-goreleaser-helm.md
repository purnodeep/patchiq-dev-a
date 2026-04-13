# ADR-009: GitHub Actions + GoReleaser + Helm for CI/CD

## Status

Accepted

## Context

PatchIQ needs to build agent binaries for 6 platform/arch combinations, Docker images, and Helm charts from a single pipeline.

## Decision

Use GitHub Actions for CI/CD, GoReleaser for cross-compilation and release automation, and Helm for Kubernetes deployment.

## Consequences

- **Positive**: Cross-compiles 6 agent binaries + Docker images in one pipeline; GoReleaser handles checksums and signing; Helm chart for K8s customers; GitHub-native (no external CI service)
- **Negative**: GitHub Actions has limited self-hosted runner support for macOS; GoReleaser config can be complex; Helm chart maintenance is ongoing

## Alternatives Considered

- **GitLab CI**: More features but different ecosystem — rejected because we're on GitHub
- **Dagger**: Programmable CI — rejected because newer, less community support
- **Custom build scripts**: Manual cross-compilation — rejected because GoReleaser automates all of this
