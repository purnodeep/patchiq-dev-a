# Extended Agent Collectors

**Status**: Planned
**Wave**: 1 — Foundation Polish
**Dependencies**: Module registry, run_scan enhancements
**Consumed by**: Full Compliance Engine, Baseline Profiles

---

## Vision

Expand the agent beyond packages and hardware to provide complete endpoint visibility — security posture, running applications, energy consumption, network configuration, and certificates.

## Deliverables

### Built-in Security Collectors
- [ ] Antivirus status: product name, version, definition date, real-time protection state (Win: WMI, Mac: XProtect/MDM, Linux: ClamAV/custom)
- [ ] Firewall state: enabled/disabled, active rules count, profile (Win: netsh, Mac: pfctl, Linux: iptables/nftables/ufw)
- [ ] Disk encryption: BitLocker/FileVault/LUKS status per volume, encryption percentage
- [ ] VPN/Proxy: active VPN connections, proxy configuration, connected network
- [ ] USB policy: connected USB devices, storage policy (allow/block)
- [ ] Screen lock: lock timeout setting, screensaver state

### Application Collectors
- [ ] Running applications: process list with resource usage, listening ports, startup items
- [ ] Installed applications (beyond APT): Flatpak, Snap, AppImage, `/opt/` scanner, `.desktop` file parser
- [ ] npm/pip/gem global packages
- [ ] macOS: Homebrew casks, DMG-installed apps in /Applications
- [ ] Windows: Add/Remove Programs (registry), MSIX/AppX, Chocolatey, winget

### Infrastructure Collectors
- [ ] Energy metrics: battery health, power consumption, CPU power state (Mac/Linux)
- [ ] Network configuration: DNS servers, gateway, DHCP/static, WiFi SSID
- [ ] Certificate store: installed certs, expiry dates, trust chain (Win/Mac)

### Scan Type System
- [ ] `SCAN_TYPE_FULL`: all collectors
- [ ] `SCAN_TYPE_QUICK`: packages only (existing)
- [ ] `SCAN_TYPE_TARGETED`: specify categories via `check_categories` field
- [ ] Collector enable/disable via agent settings

## Architecture

Each collector registers as a sub-collector within the inventory module following existing OS-specific build tag pattern (`_linux.go`, `_darwin.go`, `_windows.go`).

## License Gating
- Basic inventory (packages, hardware, OS): all tiers
- Extended security collectors: PROFESSIONAL+
- Energy and usage monitoring: ENTERPRISE
- Custom collector scheduling: ENTERPRISE
