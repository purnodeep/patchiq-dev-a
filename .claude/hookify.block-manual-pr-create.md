---
name: block-manual-pr-create
enabled: true
event: bash
pattern: gh\s+pr\s+create
action: warn
---

**BLOCKED: Manual `gh pr create` detected.**

Use `/commit-push-pr` instead. It commits, pushes, and creates the PR with a properly formatted description (Summary + Test Plan) in one flow.

See CLAUDE.md → Development Workflow → Shipping.
