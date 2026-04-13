# Audit 12: Documentation Quality, Stale Artifacts, and Project Clutter

**Date**: 2026-04-09
**Branch**: dev-a
**Auditor**: Claude Opus 4.6

---

## 1. Root-Level Clutter

### 1a. UI Snapshot Files — Important

**Files:**
- `/agent-overview-snapshot.md` (13.7 KB)
- `/hardware-snapshot.md` (4.8 KB)
- `/compliance-page.yml` (341 B)
- `/policy-detail-snap.md` (5.4 KB)

**Finding:** These are Playwright/accessibility-tree snapshots of UI pages — raw DOM-like dumps with `[ref=eN]` annotations. They are debugging artifacts, not documentation. They have no value checked into the repo and pollute the root directory.

**Action:** Delete all four files.

### 1b. `prototype-ui/` Directory — Important

**Path:** `/prototype-ui/` (16 files/dirs, includes `package-lock.json` at 211 KB)

**Finding:** This is an old Vite+React prototype with mock data pages for agent, hub-manager, and patch-manager. The gitignore comment says "Prototype UI mockups are committed (referenced by issues)" and git log shows it was added in commit `537665d` ("add prototype-ui mockups to repo, update issue references"). However, the actual production UIs (`web/`, `web-hub/`, `web-agent/`) have long since superseded these mockups. The `package-lock.json` (211 KB) is tracked — this should at minimum be in `.gitignore` since the project uses pnpm.

**Action:** If issue references still need the visual context, move to `docs/_archive/prototype-ui/` or a wiki. Otherwise delete. At minimum, add `prototype-ui/package-lock.json` to `.gitignore` and remove from tracking.

---

## 2. Stale Docs

### 2a. `docs/_revision/` — Broken Redirects — Important

**Path:** `/docs/_revision/` (4 files)

All four files are "MOVED" stubs pointing to docs that no longer exist:
- `development-process.md` -> `docs/DEVELOPMENT-PROCESS.md` — **target does not exist**
- `roadmap.md` -> `docs/roadmap.md` — **target does not exist**
- `project-structure.md` -> `docs/blueprint/core/project-structure.md` — target exists (OK)
- `tech-stack.md` -> `docs/blueprint/core/tech-stack.md` — target exists (OK)

**Finding:** CLAUDE.md references `docs/roadmap.md`, `docs/DEVELOPMENT-PROCESS.md`, `docs/PM.md`, and `docs/PROJECT.md` in the "Key Documentation" table — **none of these files exist**. The revision stubs point to the same missing targets. This means the CLAUDE.md "Key Documentation" table is partially stale.

**Action:**
1. Either create the missing docs or update CLAUDE.md to point to the files that actually exist (likely the blueprint equivalents).
2. Delete or update the `_revision/` stubs that point to missing targets.

### 2b. `docs/_archive/BLUEPRINT-V1.md` — Minor

**Path:** `/docs/_archive/BLUEPRINT-V1.md` (32 KB), `/docs/_archive/roadmap-v1.md` (2.8 KB)

**Finding:** Archived docs from V1. The `_archive/` directory is in `.gitignore` with the pattern `docs/_archive/` but these files are still tracked (they were committed before the gitignore rule was added). They consume 35 KB of tracked space.

**Action:** Run `git rm --cached docs/_archive/` to untrack them. The gitignore rule will prevent re-adding.

### 2c. CLAUDE.md "Key Documentation" Table — Important

**Finding:** The CLAUDE.md Key Documentation table lists:
- `docs/roadmap.md` — does not exist
- `docs/DEVELOPMENT-PROCESS.md` — does not exist
- `docs/PM.md` — does not exist
- `docs/PROJECT.md` — does not exist

These are core onboarding docs. A new developer following CLAUDE.md would immediately hit four 404s.

**Action:** Update the table to reference the actual file locations, or create the files.

---

## 3. ADR Status

### 3a. No ADR Has a Status Value — Important

**Finding:** All 24 ADRs have a `## Status` heading but **none have an actual status value** (e.g., "Accepted", "Superseded", "Proposed"). The only exception is ADR-008 which has "Accepted (Updated — Redis replaced with Valkey per ADR-011)" as body text below the heading. ADRs 013 and 014 have context paragraphs where the status value should be.

Per standard ADR conventions, every ADR should have one of: Proposed, Accepted, Deprecated, Superseded.

**Action:** Add status values to all ADRs. At minimum:
- ADR-008: Already has a status (good, just inconsistent formatting)
- ADR-009: Mentions Helm, but no Helm charts exist in the repo — should note "Helm: not yet implemented"
- All others: Add "Accepted" or appropriate status

### 3b. ADR-009 References Helm — Minor

**Path:** `/docs/adr/009-github-actions-goreleaser-helm.md`

**Finding:** ADR title includes "Helm" and the decision references Helm charts for Kubernetes deployment, but no Helm charts exist anywhere in the repo. This is either future work or an abandoned plan.

**Action:** Add a note to the ADR clarifying Helm is not yet implemented (M3/M4 scope).

---

## 4. Plan Status

### 4a. `docs/plans/POC-PLAN.md` — Minor

**Path:** `/docs/plans/POC-PLAN.md` (34 KB)

**Finding:** A large POC deployment plan. No date prefix like the other plans. Unclear if this is active, completed, or abandoned. All other plans follow `YYYY-MM-DD-*` naming.

**Action:** Add a date prefix and a status header (Active/Completed/Abandoned).

### 4b. `docs/superpowers/plans/` — 364 KB of Generated Plans — Minor

**Path:** `/docs/superpowers/plans/` (7 files, 364 KB), `/docs/superpowers/specs/` (5 files, 76 KB)

**Finding:** These are brainstorming/planning outputs from the Superpowers plugin (Claude Code agent tooling). They are generated artifacts tied to specific implementation sessions. Twelve of these files are tracked in git. They add 440 KB of session-specific content that is not useful as project documentation.

**Action:** Consider adding `docs/superpowers/` to `.gitignore` and untracking. These are ephemeral planning artifacts, not permanent docs.

---

## 5. Dead Directories

### 5a. `templates/reports/` — Empty — Minor

**Path:** `/templates/reports/` (only `.gitkeep`)

**Finding:** Empty directory placeholder. No report templates exist. The `templates/` directory has no other content.

**Action:** Keep if report templates are planned for near-term. Remove if not on the M2/M3 roadmap.

### 5b. `tools/` — Empty — Minor

**Path:** `/tools/` (only `.gitkeep`)

**Finding:** Empty directory placeholder. There is a `tools.go` at root level that imports Go tool dependencies (sqlc, goose, buf, etc.) which is standard Go practice. The `tools/` directory itself is unused.

**Action:** Remove if no tool scripts are planned. The Go tool dependencies live in `tools.go`, not here.

### 5c. `test/e2e/` and `test/load/` — Empty — Minor

**Paths:** `/test/e2e/` (only `.gitkeep`), `/test/load/` (only `.gitkeep`)

**Finding:** Empty placeholder directories. E2E and load testing frameworks are not set up yet.

**Action:** Keep if these are planned for near-term. Low priority.

---

## 6. Gitignore Gaps

### 6a. `.superpowers/` Tracked Despite Gitignore — Critical

**Finding:** `.superpowers/` is listed in `.gitignore` but **31 files are still tracked in git**. These include `.server.log`, `.server.pid`, `.events`, and HTML brainstorming artifacts. PID files and server logs should never be in version control.

**Action:** Run `git rm -r --cached .superpowers/` to untrack all files. The existing gitignore rule will prevent re-adding.

### 6b. `Audit/` Directory Not Gitignored — Important

**Finding:** The `Audit/` directory (where this file lives) is not in `.gitignore` and not tracked in git. If audit reports are meant to be temporary, add `Audit/` to `.gitignore`. If they should be committed, that is fine but should be a deliberate choice.

**Action:** Add `Audit/` to `.gitignore` or explicitly commit audit reports.

### 6c. `.omniprod/` and `.omniprod-plugin/` Not Gitignored — Important

**Finding:** 30 files in `.omniprod/` and 45 files in `.omniprod-plugin/` are tracked. These include:
- `.omniprod/findings/` — JSON finding reports (234 KB total)
- `.omniprod/reviews/` — Markdown review reports
- `.omniprod/product-map.json` (89 KB)
- `.omniprod-plugin/` — Plugin source code and config

The `.omniprod/` directory appears to be a product compliance/QA tool. The `findings/` and `reviews/` subdirectories contain session-specific outputs. If this is a third-party plugin, it should probably be gitignored or vendored properly.

**Action:** Evaluate whether `.omniprod/` findings and reviews should be tracked. If they are ephemeral QA outputs, add to `.gitignore` and untrack.

### 6d. `.claude/projects/-home-heramb-skenzeriq-patchiq/` — Stale Path — Minor

**Finding:** Three Claude project memory files are tracked under a path referencing `/home/heramb/skenzeriq/patchiq/` — an old directory structure. These are OmniProd review results (compliance, endpoints, patches). The path no longer matches the current project location.

**Action:** These are stale. Remove from tracking or move to the correct project path.

### 6e. `prototype-ui/package-lock.json` Tracked — Minor

**Finding:** The project uses pnpm (has `pnpm-lock.yaml` and `pnpm-workspace.yaml`). The `prototype-ui/` has a separate `package-lock.json` (npm lockfile, 211 KB) tracked in git. This is inconsistent and wastes space.

**Action:** Add `prototype-ui/package-lock.json` to `.gitignore` and untrack.

---

## 7. README / Getting Started

### 7a. No Root README — Critical

**Finding:** There is **no README.md at the project root**. A developer cloning this repo gets zero orientation. CLAUDE.md serves as the AI agent guide but is not a substitute for a human-readable README.

A README should cover: project description, prerequisites, quickstart (`make dev`), architecture overview, link to docs.

**Action:** Create a `README.md` at project root with at minimum: project name/description, prerequisites (Go 1.25, Node 20, pnpm 9, Docker), quickstart instructions, architecture diagram reference, and links to `CLAUDE.md` and `docs/blueprint/`.

### 7b. `.env.example` Missing Port Offset Documentation — Minor

**Path:** `/.env.example`

**Finding:** The `.env.example` shows hardcoded default ports (8080, 8090, 8082) but does not mention the per-user port offset system documented in CLAUDE.md. A new developer might manually create `.env` from this example instead of using `make dev-env`.

**Action:** Add a comment to `.env.example`: "# Do not edit manually. Run 'make dev-env' to generate with correct per-user port offsets."

---

## 8. Tooling Configs

### 8a. `openapitools.json` — OK (Used)

**Finding:** Referenced by `make api-client` in Makefile for OpenAPI spec validation. Currently in use.

**Action:** None needed.

### 8b. `skills-lock.json` — Minor

**Finding:** This is a lock file for the Claude Code skills system. It references `vercel-labs/skills`. It is a tooling artifact that probably should not be in version control (similar to how `.superpowers/` is gitignored).

**Action:** Consider adding to `.gitignore` if this is a local tooling artifact.

### 8c. `.mcp.json` — Minor

**Finding:** MCP server configuration for Chrome DevTools. This is a developer-local tool configuration. Should probably be gitignored or at least documented.

**Action:** Consider whether this should be tracked (shared team config) or gitignored (personal config).

---

## 9. Templates

### 9a. `templates/reports/` — Empty Placeholder — Minor

See item 5a above. Only contains `.gitkeep`.

### 9b. ADR Template — OK

**Path:** `/docs/adr/template.md`

**Finding:** A proper ADR template exists and is referenced by the ADR workflow.

**Action:** None needed.

---

## 10. Tools

### 10a. `tools/` Directory — Empty — Minor

See item 5b above. Only contains `.gitkeep`. Go tool dependencies are properly managed via `tools.go` at root.

### 10b. Duplicate Agent Install Scripts — Important

**Finding:** Agent install scripts exist in two locations with different versions:
- `/deploy/scripts/install-agent-macos.sh` — older version (different usage comment)
- `/scripts/install-agent-macos.sh` — newer version (more options, includes uninstall scripts)

`/deploy/scripts/` has only macOS + Windows (PowerShell). `/scripts/` has Linux + macOS + Windows install AND uninstall scripts.

**Action:** Consolidate to one location. The `/scripts/` directory has the more complete set. Remove `/deploy/scripts/install-agent-macos.sh` and `/deploy/scripts/install-agent.ps1`, or make `/deploy/scripts/` symlink to `/scripts/`.

---

## Summary

| # | Finding | Severity | Category |
|---|---------|----------|----------|
| 1a | Root-level UI snapshot files (4 files) | Important | Clutter |
| 1b | `prototype-ui/` stale mockup directory | Important | Clutter |
| 2a | `docs/_revision/` broken redirect stubs | Important | Stale docs |
| 2b | `docs/_archive/` tracked despite gitignore | Minor | Gitignore |
| 2c | CLAUDE.md Key Documentation table has 4 dead links | Important | Stale docs |
| 3a | No ADR has a status value | Important | ADR |
| 3b | ADR-009 references Helm (not implemented) | Minor | ADR |
| 4a | `POC-PLAN.md` missing date prefix and status | Minor | Plans |
| 4b | `docs/superpowers/` generated artifacts tracked | Minor | Plans |
| 5a | `templates/reports/` empty | Minor | Dead dirs |
| 5b | `tools/` empty | Minor | Dead dirs |
| 5c | `test/e2e/` and `test/load/` empty | Minor | Dead dirs |
| 6a | `.superpowers/` 31 files tracked despite gitignore | Critical | Gitignore |
| 6b | `Audit/` not gitignored | Important | Gitignore |
| 6c | `.omniprod/` + `.omniprod-plugin/` 75 files not gitignored | Important | Gitignore |
| 6d | Stale `.claude/projects/` path from old directory | Minor | Gitignore |
| 6e | `prototype-ui/package-lock.json` tracked (npm, not pnpm) | Minor | Gitignore |
| 7a | No root README.md | Critical | README |
| 7b | `.env.example` missing port offset docs | Minor | README |
| 8b | `skills-lock.json` tracked | Minor | Tooling |
| 8c | `.mcp.json` tracked (local tool config) | Minor | Tooling |
| 10b | Duplicate agent install scripts in two dirs | Important | Tools |

**Totals:** 2 Critical, 8 Important, 12 Minor
