# Remote Access (RDP / Terminal / File)

**Status**: Planned
**Milestone**: M4
**Dependencies**: M1 agent gRPC comms, M2 RBAC, M3 extended collectors, MFA (M3 or prerequisite)

---

## Vision

Allow operators to open a terminal, remote desktop session, or file transfer directly to a managed endpoint through the PatchIQ UI, with full auditability and approval controls.

## Deliverables

### Browser Terminal
- [ ] WebSocket endpoint on Patch Manager server proxies to agent pseudo-terminal via gRPC
- [ ] Agent opens PTY on request; streams stdin/stdout/stderr bidirectionally
- [ ] Terminal UI in web/ (xterm.js); resize events forwarded to PTY
- [ ] Session token scoped to single connection; expires on disconnect

### Remote Desktop
- [ ] Screen capture: agent captures framebuffer at configurable FPS, streams as JPEG/WebP frames
- [ ] Input forwarding: keyboard and mouse events sent from browser to agent
- [ ] Browser viewer using HTML5 canvas; adaptive quality based on bandwidth
- [ ] Windows (RDP-over-gRPC) and Linux (X11/Wayland screenshot API) targets

### File Transfer
- [ ] Download: retrieve log files, config files, diagnostic bundles from agent
- [ ] Upload: push config updates, scripts, patches to agent staging directory
- [ ] File browser UI: directory listing, size, permissions, last modified
- [ ] Transfer progress and integrity check (SHA-256 on both ends)

### Security Controls
- [ ] Session RBAC: separate permissions — `remote:terminal`, `remote:desktop`, `remote:file`
- [ ] Approval workflow: high-privilege sessions require second-user approval (configurable)
- [ ] Time limits: 30-minute default session cap; configurable per policy
- [ ] MFA challenge on session initiation (TOTP or Zitadel MFA)
- [ ] Full session recording: terminal replay (asciinema format), desktop video archive
- [ ] Audit event emitted on session start, end, and each file transfer

## License Gating

- Browser terminal: ENTERPRISE
- Remote desktop: ENTERPRISE
- File transfer: ENTERPRISE
