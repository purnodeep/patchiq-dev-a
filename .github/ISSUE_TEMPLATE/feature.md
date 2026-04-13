---
name: Feature
about: New feature or enhancement (Standard Track)
title: ''
labels: feature
assignees: ''
---

## What
<!-- 1-2 sentences: what needs to be built -->

## Why
<!-- 1-2 sentences: what problem this solves -->

## Acceptance Criteria
- [ ] <!-- Specific, testable criterion -->
- [ ] <!-- Another criterion -->
- [ ] Tests pass: `make ci`

## Workflow

Track: **Standard**

1. `/brainstorm` — design exploration
2. `/write-plan` — granular implementation plan
3. Worktree (auto via skill)
4. TDD: failing test → implement → pass (auto via skill)
5. `/review-pr all parallel` — fix all Critical/Important
6. `/commit-push-pr` — create PR
7. Request review from @herambskanda

<!-- For UI features, add: Enable `frontend-design` plugin before starting -->

## Files to Touch
- <!-- `path/to/file` — what changes -->

## Reference
- <!-- Link to similar existing implementation -->

## Out of Scope
- <!-- What this issue does NOT cover -->

## Dependencies
- <!-- #issue-number — why this must be done first -->
<!-- If no dependencies, write "None" -->

## Blocks
- <!-- #issue-number — what this unblocks when done -->
<!-- If nothing blocked, write "None" -->

## Labels Checklist
- [ ] Type label added (feature/enhancement)
- [ ] Priority label added (P0-P3)
- [ ] Risk label added (risk:critical/high/medium/low)
