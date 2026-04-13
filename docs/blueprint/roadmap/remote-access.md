# Remote Access (RDP)

**Status**: Planned
**Milestone**: M4
**Dependencies**: Agent system module, RBAC permissions, WebSocket infrastructure

---

## Vision

Allow administrators with sufficient RBAC permissions to remotely access and control managed endpoints directly through the PatchIQ web interface. The agent acts as a secure gateway, enabling remote desktop, terminal, and file transfer sessions without requiring separate VPN or RDP infrastructure.

## User Value

- **Troubleshooting**: Diagnose patch failures, compliance violations, or endpoint issues in real-time
- **Remediation**: Manually fix issues that automated scripts cannot handle
- **Audit**: All remote sessions are logged with video recording (optional) for compliance
- **Zero infrastructure**: No separate RDP/VNC/SSH setup — everything through PatchIQ
- **Secure by default**: Sessions are RBAC-gated, time-limited, and audited

## Architecture

### Session Flow

```
Admin (Browser)
    │ WebSocket (wss://)
    ▼
Patch Manager Server
    │ gRPC bidirectional stream
    ▼
Agent (Endpoint)
    │ Local session
    ▼
Desktop / Terminal / File System
```

### Session Types

| Type | Protocol | Use Case |
|------|----------|----------|
| **Terminal** | WebSocket → agent runs pseudo-terminal | CLI troubleshooting, script execution |
| **Desktop** | WebSocket → agent captures screen, forwards input | GUI-based troubleshooting, verification |
| **File Transfer** | WebSocket → agent reads/writes files | Log retrieval, config file updates |

### Security Model

- **RBAC**: `remote:terminal`, `remote:desktop`, `remote:file_transfer` permissions (separate)
- **Approval workflow**: Optional — require manager approval before session starts (workflow `approval` node)
- **Time limits**: Sessions auto-terminate after configurable duration (default: 30 min)
- **Recording**: Terminal sessions logged as text, desktop sessions optionally recorded
- **Audit trail**: Session start, end, commands executed, files transferred — all in audit log
- **MFA**: Require re-authentication before starting a remote session

### Agent Side

The `system` module extends with a `remote_session` command:
- Opens a local pseudo-terminal (terminal sessions)
- Starts screen capture + input forwarding (desktop sessions — platform-specific)
- Streams data over the gRPC connection back to server
- Enforces session timeout and resource limits

## Foundations Built in M2

- **Agent `system` module**: Extensible module for system-level operations
- **RBAC permission model**: Fine-grained permissions per action
- **gRPC bidirectional streams**: Existing heartbeat/sync infrastructure
- **Audit logging**: All actions emit domain events to audit table

## License Gating

- Remote terminal: ENTERPRISE only
- Remote desktop: ENTERPRISE only
- File transfer: ENTERPRISE only
- Session recording: ENTERPRISE only
