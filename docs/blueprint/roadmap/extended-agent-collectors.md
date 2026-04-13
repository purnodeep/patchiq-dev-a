# Extended Agent Collectors

**Status**: Planned
**Milestone**: M3
**Dependencies**: Module registry, `run_scan` enhancements
**Consumed by**: Compliance Engine v2 (uses collector data for compliance rule evaluation)

---

## Vision

Expand the agent's data collection beyond packages and hardware to provide a comprehensive endpoint inventory — usage data, running applications, energy consumption, security posture, and any metric the compliance engine or admin needs.

## User Value

- Complete endpoint visibility: not just "what's installed" but "what's running, how it's configured, how it's being used"
- Feed compliance rules with real data (antivirus status, firewall state, encryption status)
- Capacity planning with usage and resource consumption metrics
- Energy monitoring for sustainability compliance and cost management
- Application inventory for license management and shadow IT detection

## Planned Collectors

### Built-in Collectors (ship with agent)

| Collector | Platform | Data |
|-----------|----------|------|
| **Antivirus status** | Win/Mac/Linux | Product name, version, definition date, real-time protection state |
| **Firewall state** | Win/Mac/Linux | Enabled/disabled, active rules count, profile (domain/private/public on Windows) |
| **Disk encryption** | Win/Mac/Linux | BitLocker/FileVault/LUKS status per volume, encryption percentage |
| **VPN/Proxy** | All | Active VPN connections, proxy configuration, connected network |
| **USB policy** | Win/Mac/Linux | Connected USB devices, storage policy (allow/block) |
| **Screen lock** | Win/Mac | Lock timeout setting, screensaver state |
| **Running applications** | All | Process list with resource usage, listening ports, startup items |
| **Energy metrics** | Mac/Linux | Battery health, power consumption, CPU power state |
| **Network configuration** | All | DNS servers, gateway, DHCP/static, WiFi SSID |
| **Certificate store** | Win/Mac | Installed certificates, expiry dates, trust chain |

### Agent Module Structure

Each collector category registers as a sub-collector within the inventory module:

```go
type SecurityCollector struct{}
func (c *SecurityCollector) Name() string { return "security" }
func (c *SecurityCollector) Collect(ctx context.Context) (*SecurityReport, error) {
    // antivirus, firewall, encryption, USB policy
}
```

Collectors are OS-specific (build tags: `_linux.go`, `_darwin.go`, `_windows.go`) following the existing pattern in `internal/agent/inventory/`.

### Scan Types Integration

The `RunScanPayload.check_categories` field controls which collectors run:
- `["packages"]` → quick scan (existing)
- `["packages", "services", "security"]` → targeted scan
- `[]` (empty) or `SCAN_TYPE_FULL` → everything

## Foundations Built in M2

- **Module registry**: Collectors register via the same module pattern
- **`run_scan` scan types**: `ScanType` enum and `check_categories` field defined
- **Inventory report**: Extended to carry new data categories
- **Settings watcher**: Controls scan intervals and collector enable/disable

## License Gating

- Basic inventory (packages, hardware, OS): all tiers
- Extended security collectors: PROFESSIONAL+
- Energy and usage monitoring: ENTERPRISE
- Custom collector scheduling: ENTERPRISE
