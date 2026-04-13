# Script-based Custom Collectors

**Status**: Planned
**Milestone**: M3
**Dependencies**: Executor module, script library, `run_script` command
**Consumed by**: Compliance Engine v2 (uses script collector output for custom compliance rules)

---

## Vision

Allow administrators to define custom data collection scripts that run on agents, extending endpoint visibility beyond built-in collectors. Any data point the organization cares about — proprietary application status, internal service health, custom security checks, environmental metrics — becomes collectible without an agent update.

## User Value

- **Infinite extensibility**: Collect any data point without waiting for a PatchIQ release
- **Compliance flexibility**: Define checks for organization-specific requirements (proprietary VPN, custom antivirus, internal PKI)
- **No agent updates**: New collection capabilities deployed via script push, not binary update
- **Structured results**: Scripts return structured data that feeds into compliance rules, dashboards, and alerts
- **Reusable library**: Organization builds a library of collection scripts, shareable across policies and frameworks

## Architecture

### Script Definition

Admins define collector scripts in the script library:

```json
{
  "name": "check-corporate-vpn",
  "description": "Verify corporate VPN client is installed and connected",
  "script_type": "shell",
  "script": "#!/bin/bash\nif pgrep -x 'vpnclient' > /dev/null; then\n  echo '{\"status\": \"connected\", \"server\": \"vpn.corp.com\"}'\nelse\n  echo '{\"status\": \"disconnected\"}'\nfi",
  "expected_output": "json",
  "timeout_seconds": 30,
  "platforms": ["linux", "darwin"],
  "tags": ["security", "compliance"]
}
```

### Collection Schedule

Scripts can be scheduled to run:
- **Periodically**: Every N hours (configurable per script)
- **On-demand**: As part of a `run_scan` command with `SCAN_TYPE_TARGETED`
- **Event-driven**: Workflow trigger fires the collection
- **Compliance-driven**: Compliance rule references the script, runs on evaluation

### Data Flow

```
Script Library (Server)
    │ push via run_script command
    ▼
Agent (executor module)
    │ runs script, captures output
    ▼
Outbox → SyncOutbox → Server
    │ parse structured output
    ▼
Endpoint data store
    │
    ├── Compliance engine: evaluate against rules
    ├── Dashboard: display in endpoint detail
    └── Alerts: trigger on anomalies
```

### Output Format

Scripts must return JSON for structured processing:
```json
{
  "check": "corporate-vpn",
  "status": "pass",
  "data": {
    "connected": true,
    "server": "vpn.corp.com",
    "protocol": "IKEv2",
    "uptime_hours": 47
  }
}
```

The server validates output against the script's `expected_output` schema. Non-JSON output is stored as raw text (still usable for compliance pass/fail based on exit code).

### Script Library Management

- CRUD API: `POST /api/v1/scripts`, `GET /api/v1/scripts`, etc.
- Version control: scripts are versioned, previous versions retained
- Platform targeting: scripts declare compatible platforms
- Testing: dry-run execution on selected endpoints before publishing
- Sharing: scripts can be exported/imported across Patch Manager instances

## Foundations Built in M2

- **`executor` module**: Agent-side script execution with timeout, output limits, temp file cleanup
- **`RunScriptPayload`**: Proto message with script type, timeout, env vars, output limits
- **`run_script` command**: End-to-end server-to-agent script execution pipeline
- **Script library reference**: `script_id` field in `RunScriptPayload` for library-stored scripts
- **Workflow `script` node**: Integrates script execution into workflow DAGs

## License Gating

- Script library management: PROFESSIONAL+
- Scheduled script collection: PROFESSIONAL+
- Custom collector for compliance rules: ENTERPRISE
- Script export/import: ENTERPRISE
