---
name: warn-stop-verification
enabled: true
event: stop
pattern: .*
action: warn
---

**Before finishing, verify:**

1. Did you follow the correct workflow track (Standard / Fix / Quick)?
2. Did you run `/review-pr` before creating the PR?
3. Did you use the `verification-before-completion` skill to confirm tests pass?
4. Did you use `/commit-push-pr` (not manual git commands)?

If any answer is NO, go back and complete the missing step.
