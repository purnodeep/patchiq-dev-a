# Script-based Custom Collectors

**Status**: Planned
**Wave**: 2 — Automation & Extensibility
**Dependencies**: Executor module, run_script command
**Consumed by**: Full Compliance Engine (custom compliance rules)

---

## Vision

Custom data collection scripts that run on agents, extending visibility beyond built-in collectors. Any data point — proprietary app status, internal service health, custom security checks — becomes collectible without an agent update.

## Deliverables

### Script Library
- [ ] CRUD API: `POST /api/v1/scripts`, versioning, platform targeting
- [ ] Script types: shell, PowerShell, Python
- [ ] Expected output: JSON (structured) or text (exit-code based)
- [ ] Timeout and resource limits per script

### Collection Scheduling
- [ ] Periodic: every N hours per script
- [ ] On-demand: via `run_scan` with `SCAN_TYPE_TARGETED`
- [ ] Event-driven: workflow trigger fires collection
- [ ] Compliance-driven: compliance rule references script

### Output Processing
- [ ] Structured JSON output feeds into compliance rules, dashboards, alerts
- [ ] Non-JSON stored as raw text (pass/fail via exit code)
- [ ] Server-side schema validation against expected output format

### Operations
- [ ] Dry-run: test on selected endpoints before publishing
- [ ] Import/export across Patch Manager instances
- [ ] Script execution history and output logs

## License Gating
- Script library management: PROFESSIONAL+
- Scheduled collection: PROFESSIONAL+
- Compliance rule integration: ENTERPRISE
- Import/export: ENTERPRISE
