---
name: block-manual-git-commit
enabled: true
event: bash
pattern: git\s+commit
action: warn
---

**BLOCKED: Manual git commit detected.**

Use `/commit` or `/commit-push-pr` instead. These commands auto-generate proper commit messages with Co-Authored-By and follow the project's commit conventions.

See CLAUDE.md → Development Workflow → Shipping.
