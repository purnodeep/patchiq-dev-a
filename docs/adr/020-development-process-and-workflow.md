# ADR-020: Development Process, Git Strategy, and Plugin-Enforced Workflows

## Status

Proposed

## Context

PatchIQ development has operated with an immature process: all developers SSH into a single shared server, commit directly to one GitHub account with no branch strategy, no PR reviews, no issue tracking discipline, and no contribution guidelines. The result is untraceable commits, conflicting changes, no code review, and zero accountability.

The team fluctuates between 2–7 full-stack developers. All developers are paired with Claude Code and have access to 17 Anthropic-official plugins covering the full development lifecycle — from brainstorming through code review to deployment.

We need a development process that:
1. Establishes clear developer identity and accountability (who changed what)
2. Protects the main branch from unreviewed code
3. Enforces a systematic, plugin-driven workflow for every type of work
4. Scales from 2 to 7 developers without coordination overhead
5. Leverages Claude Code plugins as the enforcement layer — not just guidelines on paper

## Decision

### 1. Developer Environment: Per-User Isolation on Shared Server

Each developer gets their own Unix user account on the shared development server with:
- Personal SSH key pair (for both server access and GitHub authentication)
- Personal git configuration (`user.name`, `user.email`) — every commit is attributable
- Personal clone of the repository in their home directory
- Personal Claude Code session with all 17 plugins installed

No shared Unix accounts. No shared git identities. No shared Claude Code sessions.

### 2. Git Strategy: Trunk-Based Development with Short-Lived Feature Branches

- **Protected `main` branch** — no direct pushes, all changes via squash-merge PRs
- **Short-lived feature branches** — max 3 days, branched from latest `main`
- **Branch naming**: `{type}/{issue-id}-{short-description}` (e.g., `feat/PIQ-42-visual-workflow-builder`, `fix/PIQ-99-grpc-reconnect`)
- **Types**: `feat/`, `fix/`, `refactor/`, `docs/`, `chore/`, `test/`
- **Squash merge** to main — clean linear history, one commit per PR
- **Git worktrees** (enforced by superpowers plugin) for workspace isolation during development
- **Branch cleanup** via `/clean_gone` after merges

### 3. GitHub as Single Source of Truth

- **GitHub Issues** for all work items (features, bugs, tasks, tech debt)
- **GitHub Projects** for sprint boards
- **GitHub PRs** for all code changes — no exceptions
- **PR template** enforced: What changed, Why, How to test, Screenshots (if UI)
- **At least 1 human reviewer** required before merge
- **CI must pass** (lint, unit tests, integration tests, build) before merge

### 4. Plugin-Enforced Development Lifecycle

Every piece of work follows a specific plugin-driven workflow based on its type. The plugins are not optional tooling — they ARE the process.

See the Development Process Guide (`docs/DEVELOPMENT-PROCESS.md`) for the complete step-by-step workflows.

### 5. Sprint Cadence and Project Management

- **2-week sprints** planned with `/sprint-planning`
- **Daily standups** generated with `/standup`
- **Stakeholder updates** via `/stakeholder-update` at sprint boundaries
- **Roadmap reviews** via `/roadmap-update` monthly
- **Architecture decisions** documented as ADRs via `/architecture`

## Consequences

### What becomes easier
- Every commit is attributable to a specific developer
- Code quality is enforced by 6 specialized review agents before any PR merges
- TDD is non-negotiable — the superpowers plugin blocks code-first approaches
- New developers onboard by reading one document and following the plugin-enforced workflow
- Sprint planning and stakeholder communication are standardized
- Security vulnerabilities are caught at edit-time by the security-guidance hook
- Library documentation is always current via context7

### What becomes harder
- Quick hotfixes require the full branch → PR → review → merge flow (this is intentional)
- Developers must learn the plugin commands and trust the workflow
- Initial overhead for per-user server setup
- The brainstorm → plan → implement pipeline adds time upfront (saves time overall)

### What we'll need to revisit
- Sprint cadence (2 weeks) may need adjustment once velocity is established
- CI pipeline specifics (test containers, build matrix) as the codebase grows
- Whether to move from shared server to individual dev machines or cloud dev environments

## Alternatives Considered

### Gitflow (long-lived develop branch)
Rejected — too complex for 2–7 developers. Trunk-based with short-lived branches gives the same safety with less ceremony. Gitflow's release branches are unnecessary when we use tag-based releases with GoReleaser.

### No Plugin Enforcement (guidelines-only)
Rejected — the current mess proves that guidelines without enforcement don't work. Plugins provide mechanical enforcement: the security hook blocks insecure edits, TDD skill blocks code-before-tests, review agents block unreviewed merges.

### Individual Developer Machines (no shared server)
Considered but deferred — the shared server is the current infrastructure. Per-user accounts solve the immediate identity/accountability problem. Moving to individual machines is a future option.

### Monorepo vs Multi-repo
Not in scope — already decided (ADR-019). The monorepo structure is established.
