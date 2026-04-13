---
name: Bug Report
about: Report a bug (Fix Track)
title: 'Fix: '
labels: bug
assignees: ''
---

## Bug Description
<!-- What's broken? -->

## Steps to Reproduce
1. <!-- Step 1 -->
2. <!-- Step 2 -->
3. <!-- Step 3 -->

## Expected Behavior
<!-- What should happen -->

## Actual Behavior
<!-- What happens instead -->

## Environment
- OS: <!-- e.g., Ubuntu 24.04 -->
- Go version: <!-- e.g., 1.23.x -->
- Node version: <!-- e.g., 24.x -->
- Browser: <!-- if UI bug -->

## Acceptance Criteria
- [ ] Bug no longer reproduces with steps above
- [ ] Regression test added
- [ ] No other tests broken
- [ ] `make ci` passes

## Workflow

Track: **Fix**

1. Reproduce the bug (steps above)
2. Systematic debugging (auto via skill — 4-phase root cause analysis)
3. Write failing test → fix → verify no regressions
4. `/review-pr all parallel` — fix Critical/Important
5. `/commit-push-pr` — create PR
6. Request review from @herambskanda

> If 3+ fix attempts fail, STOP. The problem is architectural. Discuss with the team.

## Dependencies
- <!-- #issue-number — why this must be done first -->
<!-- If no dependencies, write "None" -->

## Blocks
- <!-- #issue-number — what this unblocks when done -->
<!-- If nothing blocked, write "None" -->

## Files to Touch
- <!-- `path/to/file` — what changes -->

## Reference
- <!-- Related code, docs, or previous issues -->

## Logs / Screenshots
<!-- Paste relevant logs or screenshots -->

## Labels Checklist
- [ ] Type label: `bug`
- [ ] Priority label added (P0-P3)
- [ ] Risk label added (risk:critical/high/medium/low)
