---
name: block-manual-push
enabled: true
event: bash
pattern: git\s+push
action: warn
---

**BLOCKED: Manual git push detected.**

Use `/commit-push-pr` which handles commit + push + PR creation in one flow. If you just need to push without a PR, explain why to the user first.

See CLAUDE.md → Development Workflow → Shipping.
