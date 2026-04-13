---
name: Task / Chore
about: Documentation, chores, scaffolding, quick fixes (Quick Track)
title: ''
labels: chore
assignees: ''
---

## What
<!-- 1-2 sentences: what needs to be done -->

## Why
<!-- 1-2 sentences: why this matters -->

## Acceptance Criteria
- [ ] <!-- Specific, testable criterion -->
- [ ] `make ci` passes

## Workflow

Track: **Quick**

1. Worktree (auto via skill)
2. Do the work
3. `/review-pr code errors` — targeted review
4. `/commit-push-pr` — create PR
5. Request review from @herambskanda

<!-- For scaffolding where TDD doesn't apply: implement → verify builds/lints. Note in PR why TDD was skipped. -->

## Dependencies
- <!-- #issue-number — why this must be done first -->
<!-- If no dependencies, write "None" -->

## Blocks
- <!-- #issue-number — what this unblocks when done -->
<!-- If nothing blocked, write "None" -->

## Files to Touch
- <!-- `path/to/file` — what changes -->

## Reference
- <!-- Link to similar existing implementation or pattern -->

## Labels Checklist
- [ ] Type label added (chore/docs/tech-debt)
- [ ] Priority label added (P0-P3)
- [ ] Risk label added (risk:critical/high/medium/low)
