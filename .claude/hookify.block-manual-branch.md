---
name: block-manual-branch
enabled: true
event: bash
pattern: git\s+checkout\s+-b
action: warn
---

**BLOCKED: Manual branch creation detected.**

Use the `using-git-worktrees` skill to create an isolated worktree instead. Worktrees keep your main checkout clean and prevent accidental work on the wrong branch.

See CLAUDE.md → Development Workflow → Worktree creation.
