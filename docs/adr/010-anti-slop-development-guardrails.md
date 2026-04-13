# ADR-010: Anti-Slop Development Process with AI Guardrails

## Status

Accepted

## Context

PatchIQ is developed with heavy AI assistance (Claude Code). Without strict guardrails, AI-generated code tends to drift from project conventions, add unnecessary abstractions, and introduce inconsistencies.

## Decision

Implement a multi-layer anti-slop process: CLAUDE.md as the primary AI guardrail, pre-commit hooks for automated checks, ADRs for decision documentation, and enforced PR templates with AI review.

## Consequences

- **Positive**: Strict guardrails prevent AI-generated code from drifting; ADRs ensure decisions are documented; pre-commit hooks catch issues before they reach CI; PR template enforces context sharing
- **Negative**: Initial setup overhead; developers must maintain CLAUDE.md as the project evolves; pre-commit hooks add friction to commit flow; ADR discipline requires team buy-in

## Alternatives Considered

- **CI-only checks**: No pre-commit hooks — rejected because feedback loop is too slow; developers should know immediately
- **No CLAUDE.md, rely on code review**: Human review catches everything — rejected because AI assistance is continuous and review is intermittent
- **Linter-only approach**: Just use linters — rejected because linters catch syntax, not architecture violations or convention drift
