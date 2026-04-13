---
name: warn-slog-only
enabled: true
event: file
pattern: fmt\.Print|log\.Print|log\.Fatal|log\.Panic
action: warn
---

**Use `slog` for all logging — never `fmt.Println`, `log.Println`, or `log.Printf`.**

Use structured logging: `slog.Info("message", "key", value)`

See CLAUDE.md → Code Conventions → Go.
