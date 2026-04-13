# Windows Agent Parity — Design Doc

**Date**: 2026-04-04
**Author**: Rishab + Claude
**Status**: Draft
**Track**: Standard

---

## Problem

The Windows agent (DESKTOP-629B940) sends incomplete data compared to the Linux agent (garage). The endpoint detail page in the PM UI shows dashes for most hardware fields, no network interfaces, no services, and a bare "windows" OS label. Software inventory works (147 packages), but everything else is either a stub or missing.

## Goal

Achieve full data parity between the Windows and Linux agents. Every field the Linux agent populates must have a Windows equivalent. After this work, the DESKTOP-629B940 endpoint detail page should be as complete as any Linux endpoint.

## Scope

### In Scope

1. **Enrollment enrichment** (`cmd/agent/sysinfo_windows.go`) — populate all `EndpointInfo` fields
2. **Hardware collector** (`internal/agent/inventory/hardware_windows.go`) — full implementation matching Linux
3. **Services collector** (`internal/agent/inventory/services_windows.go`) — Windows service enumeration
4. **Pre/post script fix** (`internal/agent/patcher/patcher.go`) — use `powershell` instead of `sh` on Windows
5. **Install enrichment** (`cmd/agent/cli/install.go`) — call `enrichEndpointInfo()` during install enrollment (both platforms)
6. **Windows config path** (`cmd/agent/cli/config.go`) — platform-specific `DefaultDataDir()` and `defaultConfigPath`
7. **Agent Downloads UI fix** (`web/src/pages/agent-downloads/AgentDownloadsPage.tsx`) — fix `--server-url` → `--server` flag

### Out of Scope

- Agent UI changes (web-agent/) — displays whatever data it gets
- Server-side changes — server already handles all fields; the gap is agent-side only
- Auto-service-registration from `install` — Linux doesn't do it either; keeping both platforms consistent (`install` → `service install` → `service start`)
- PM web UI changes — UI already renders all fields when data is present
- New proto fields — all existing structs (`HardwareInfo`, `ServiceInfo`, etc.) cover what we need
- Metrics changes — `metrics_windows.go` already works for heartbeat data

---

## Design

### 1. Enrollment Enrichment (`cmd/agent/sysinfo_windows.go`)

Current state: Sets `OsVersion` to `"windows/amd64"`, `IpAddresses` via `net.Interfaces()`, and `Tags["cpu_cores"]` via `runtime.NumCPU()`. Missing: OS detail, CPU model, memory, disk, kernel version.

**New implementation**: Single PowerShell call that queries `Win32_OperatingSystem`, `Win32_Processor`, and `Win32_LogicalDisk` together. Returns JSON, parsed in Go.

```
PowerShell query (combined):
  Win32_OperatingSystem → Caption, Version, BuildNumber, TotalVisibleMemorySize
  Win32_Processor → Name (first processor)
  Win32_LogicalDisk (DriveType=3) → Sum(Size)
```

**Field mapping:**

| Proto Field | Source | Example Value |
|-------------|--------|---------------|
| `OsVersion` | `Caption` | `"Microsoft Windows 11 Pro"` |
| `OsVersionDetail` | `Caption + " Build " + BuildNumber` | `"Microsoft Windows 11 Pro Build 26100"` |
| `CpuType` | `Win32_Processor.Name` | `"13th Gen Intel(R) Core(TM) i7-13700K"` |
| `MemoryBytes` | `TotalVisibleMemorySize * 1024` | `17179869184` (16 GB) |
| `HardwareModel` | `Version` (NT kernel version) | `"10.0.26100"` |
| `IpAddresses` | `net.Interfaces()` (already working) | `["192.168.1.50"]` |
| `Tags["cpu_cores"]` | `runtime.NumCPU()` (already working) | `"16"` |
| `Tags["disk_total_gb"]` | `Sum(Win32_LogicalDisk.Size) / 1GB` | `"512"` |
| `Tags["arch"]` | `runtime.GOARCH` | `"amd64"` |

This is **1 PowerShell call** at enrollment time (once per agent lifetime).

### 2. Hardware Collector (`internal/agent/inventory/hardware_windows.go`)

Current state: Empty stub returning `&HardwareInfo{}`.

**New implementation**: 10 collector functions matching Linux's structure exactly. Each uses PowerShell + CIM. All run sequentially inside `CollectHardware()`, same pattern as Linux.

#### 2.1 `collectCPU` → `Win32_Processor`

```
Query: Get-CimInstance Win32_Processor | Select-Object Name, Manufacturer, Family,
       NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed,
       L2CacheSize, L3CacheSize, Architecture, VirtualizationFirmwareEnabled
       | ConvertTo-Json
```

**Mapping to `CPUInfo`:**

| CPUInfo Field | CIM Property | Notes |
|---------------|-------------|-------|
| `ModelName` | `Name` | |
| `Vendor` | `Manufacturer` | |
| `Family` | `Family` (uint16) | Map to string via Win32_Processor family enum |
| `Architecture` | `Architecture` (uint16) | 9=x64, 12=ARM64 |
| `CoresPerSocket` | `NumberOfCores` | |
| `Sockets` | Count of processor instances | |
| `TotalLogical` | `NumberOfLogicalProcessors` | |
| `ThreadsPerCore` | `NumberOfLogicalProcessors / NumberOfCores` | Computed |
| `MaxMHz` | `MaxClockSpeed` | Already in MHz |
| `CacheL2` | `L2CacheSize` | In KB, format as string |
| `CacheL3` | `L3CacheSize` | In KB, format as string |
| `VirtType` | `VirtualizationFirmwareEnabled` | `"hyper-v"` if true |

Fields not available on Windows: `Model`, `Stepping`, `MinMHz`, `BogoMIPS`, `CacheL1d`, `CacheL1i`, `Flags`. These remain zero-valued (same as Linux when lscpu fields are missing).

#### 2.2 `collectMemory` → `Win32_OperatingSystem` + `Win32_PhysicalMemory`

```
Query 1: Get-CimInstance Win32_OperatingSystem | Select-Object
         TotalVisibleMemorySize, FreePhysicalMemory | ConvertTo-Json

Query 2: Get-CimInstance Win32_PhysicalMemory | Select-Object
         BankLabel, DeviceLocator, Capacity, MemoryType, ConfiguredClockSpeed,
         Manufacturer, SerialNumber, PartNumber, FormFactor | ConvertTo-Json
```

**Mapping to `MemoryInfo`:**

| MemoryInfo Field | Source |
|------------------|--------|
| `TotalBytes` | `TotalVisibleMemorySize * 1024` |
| `AvailableBytes` | `FreePhysicalMemory * 1024` |
| `NumSlots` | Count of `Win32_PhysicalMemory` instances |
| `DIMMs[].Locator` | `DeviceLocator` |
| `DIMMs[].BankLocator` | `BankLabel` |
| `DIMMs[].SizeMB` | `Capacity / 1048576` |
| `DIMMs[].Type` | `MemoryType` enum → string (26=DDR4, 34=DDR5) |
| `DIMMs[].SpeedMHz` | `ConfiguredClockSpeed` |
| `DIMMs[].Manufacturer` | `Manufacturer` |
| `DIMMs[].SerialNumber` | `SerialNumber` |
| `DIMMs[].PartNumber` | `PartNumber` |
| `DIMMs[].FormFactor` | `FormFactor` enum → string (8=DIMM, 12=SODIMM) |

Not available: `MaxCapacity`, `ErrorCorrection`. These remain zero-valued.

#### 2.3 `collectMotherboard` → `Win32_BaseBoard` + `Win32_BIOS`

```
Query: $b = Get-CimInstance Win32_BaseBoard | Select Manufacturer, Product, Version, SerialNumber
       $i = Get-CimInstance Win32_BIOS | Select Manufacturer, SMBIOSBIOSVersion, ReleaseDate
       @{board=$b; bios=$i} | ConvertTo-Json
```

**Mapping to `MotherboardInfo`:**

| Field | Source |
|-------|--------|
| `BoardManufacturer` | `Win32_BaseBoard.Manufacturer` |
| `BoardProduct` | `Win32_BaseBoard.Product` |
| `BoardVersion` | `Win32_BaseBoard.Version` |
| `BoardSerial` | `Win32_BaseBoard.SerialNumber` |
| `BIOSVendor` | `Win32_BIOS.Manufacturer` |
| `BIOSVersion` | `Win32_BIOS.SMBIOSBIOSVersion` |
| `BIOSReleaseDate` | `Win32_BIOS.ReleaseDate` (formatted) |

#### 2.4 `collectStorage` → `Win32_DiskDrive` + `Win32_LogicalDisk`

```
Query 1: Get-CimInstance Win32_DiskDrive | Select-Object
         DeviceID, Model, SerialNumber, Size, MediaType, InterfaceType,
         FirmwareRevision, Status, Partitions | ConvertTo-Json

Query 2: Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Select-Object
         DeviceID, Size, FreeSpace, FileSystem, VolumeName | ConvertTo-Json
```

**Mapping to `StorageDevice`:**

| Field | Source |
|-------|--------|
| `Name` | `DeviceID` (e.g., `\\.\PHYSICALDRIVE0`) |
| `Model` | `Model` |
| `Serial` | `SerialNumber` |
| `SizeBytes` | `Size` |
| `Type` | `MediaType` → "ssd"/"hdd" ("Fixed hard disk media" = hdd, "Solid state..." = ssd) |
| `FirmwareVersion` | `FirmwareRevision` |
| `Transport` | `InterfaceType` (SCSI, NVMe, USB, IDE) |
| `SmartStatus` | `Status` ("OK" → "PASSED") |
| `Partitions[].Name` | `Win32_LogicalDisk.DeviceID` (e.g., "C:") |
| `Partitions[].SizeBytes` | `Win32_LogicalDisk.Size` |
| `Partitions[].FSType` | `Win32_LogicalDisk.FileSystem` |
| `Partitions[].MountPoint` | Same as `DeviceID` ("C:") |
| `Partitions[].UsagePct` | `(Size - FreeSpace) / Size * 100` |

Not available without third-party tools: `TempCelsius` (needs smartmontools). Set to 0.

#### 2.5 `collectGPU` → `Win32_VideoController`

```
Query: Get-CimInstance Win32_VideoController | Select-Object
       Name, AdapterRAM, DriverVersion, PNPDeviceID | ConvertTo-Json
```

**Mapping to `GPUInfo`:**

| Field | Source |
|-------|--------|
| `Model` | `Name` |
| `VRAMMB` | `AdapterRAM / 1048576` |
| `DriverVersion` | `DriverVersion` |
| `PCISlot` | Extracted from `PNPDeviceID` (PCI bus portion) |

#### 2.6 `collectNetwork` → `Get-NetAdapter` + `Get-NetIPAddress`

```
Query: $adapters = Get-NetAdapter | Where-Object { $_.Status -eq 'Up' -or $_.Status -eq 'Disconnected' }
         | Select-Object Name, MacAddress, MtuSize, Status, LinkSpeed, InterfaceDescription, DriverName
       $ips = Get-NetIPAddress -ErrorAction SilentlyContinue
         | Select-Object InterfaceAlias, IPAddress, PrefixLength, AddressFamily
       @{adapters=$adapters; ips=$ips} | ConvertTo-Json -Depth 3
```

**Mapping to `NetworkInfo`:**

| Field | Source |
|-------|--------|
| `Name` | `Get-NetAdapter.Name` |
| `MACAddress` | `Get-NetAdapter.MacAddress` (reformat XX-XX → XX:XX) |
| `MTU` | `Get-NetAdapter.MtuSize` |
| `State` | `Get-NetAdapter.Status` → "up"/"down" |
| `SpeedMbps` | Parse `Get-NetAdapter.LinkSpeed` ("1 Gbps" → 1000) |
| `Type` | Classify from `InterfaceDescription` (Wi-Fi, Ethernet, virtual) |
| `Driver` | `Get-NetAdapter.DriverName` |
| `IPv4Addresses` | `Get-NetIPAddress` where `AddressFamily=2`, matched by `InterfaceAlias` |
| `IPv6Addresses` | `Get-NetIPAddress` where `AddressFamily=23`, matched by `InterfaceAlias` |

#### 2.7 `collectUSB` → `Win32_USBHub` + `Win32_PnPEntity`

```
Query: Get-CimInstance Win32_PnPEntity | Where-Object { $_.PNPDeviceID -like 'USB\*' }
       | Select-Object PNPDeviceID, Name | ConvertTo-Json
```

**Mapping to `USBDevice`:**

| Field | Source |
|-------|--------|
| `VendorID` | Extracted from `PNPDeviceID` (`USB\VID_XXXX&PID_YYYY`) |
| `ProductID` | Extracted from `PNPDeviceID` |
| `Description` | `Name` |
| `Bus` | Not directly available, set to `"USB"` |
| `DeviceNum` | Not directly available, set to `""` |

#### 2.8 `collectBattery` → `Win32_Battery`

```
Query: Get-CimInstance Win32_Battery | Select-Object
       BatteryStatus, EstimatedChargeRemaining, DesignCapacity,
       FullChargeCapacity, Chemistry | ConvertTo-Json
```

**Mapping to `BatteryInfo`:**

| Field | Source |
|-------|--------|
| `Present` | `true` if query returns result |
| `Status` | `BatteryStatus` enum → "Charging"/"Discharging"/"Full"/etc. |
| `CapacityPct` | `EstimatedChargeRemaining` |
| `EnergyFullWh` | `FullChargeCapacity / 1000` (mWh → Wh) |
| `EnergyDesignWh` | `DesignCapacity / 1000` |
| `HealthPct` | `FullChargeCapacity / DesignCapacity * 100` |
| `Technology` | `Chemistry` enum → "Li-ion"/"NiMH"/etc. |

Desktop endpoints (DESKTOP-629B940): `Win32_Battery` returns empty → `Present: false`. Same as Linux servers without batteries.

#### 2.9 `collectTPM` → `Get-Tpm`

```
Query: Get-Tpm | Select-Object TpmPresent, ManufacturerVersion | ConvertTo-Json
```

**Mapping to `TPMInfo`:**

| Field | Source |
|-------|--------|
| `Present` | `TpmPresent` |
| `Version` | `ManufacturerVersion` |

Note: `Get-Tpm` requires elevation. If it fails (non-admin), set `Present: false` and log warning.

#### 2.10 `collectVirtualization` → `Win32_ComputerSystem`

```
Query: Get-CimInstance Win32_ComputerSystem | Select-Object
       Model, HypervisorPresent | ConvertTo-Json
```

**Mapping to `VirtInfo`:**

| Field | Source |
|-------|--------|
| `IsVirtual` | `HypervisorPresent` or `Model` contains "Virtual" |
| `HypervisorType` | Inferred: "Virtual Machine" in Model → check for "VMware"/"VirtualBox"/"Hyper-V" in Model string |

### 3. Services Collector (`internal/agent/inventory/services_windows.go`)

Current state: No-op stub returning nil.

**New implementation**: Uses `Get-Service` PowerShell cmdlet.

```
Query: Get-Service | Select-Object Name, DisplayName, Status, StartType | ConvertTo-Json -Compress
```

**Mapping to `ServiceInfo`:**

| Field | Source |
|-------|--------|
| `Name` | `Name` |
| `Description` | `DisplayName` |
| `LoadState` | Always `"loaded"` (all returned services are loaded) |
| `ActiveState` | `Status`: "Running" → "active", "Stopped" → "inactive" |
| `SubState` | `Status`: "Running" → "running", "Stopped" → "dead", "Paused" → "paused" |
| `Enabled` | `StartType`: "Automatic" → true, else false |
| `Category` | Categorize by service name patterns (same approach as Linux) |

**Category mapping**: Reuse the same `categorizeService()` function from Linux with additional Windows-specific patterns:

```go
{"wuauserv", "WinDefend", "SecurityHealth"} → "Security"
{"Spooler", "BITS", "wuauserv"} → "System"
{"MSSQL", "MySQL", "postgresql"} → "Database"
{"W32Time", "Dnscache", "WinRM"} → "Network"
{"EventLog", "Winmgmt"} → "Monitoring"
```

### 4. Pre/Post Script Fix (`internal/agent/patcher/patcher.go`)

Current state: Line 168 uses `sh -c` for pre/post scripts on all platforms.

**Fix**: Platform-aware shell selection.

```go
// In patcher.go, replace:
preResult, err := m.executor.Execute(ctx, "sh", "-c", payload.PreScript)

// With:
shell, flag := scriptShell()
preResult, err := m.executor.Execute(ctx, shell, flag, payload.PreScript)
```

New file `internal/agent/patcher/shell_windows.go`:
```go
func scriptShell() (string, string) {
    return "powershell.exe", "-NoProfile -NonInteractive -Command"
}
```

New file `internal/agent/patcher/shell_unix.go` (build tag `!windows`):
```go
func scriptShell() (string, string) {
    return "sh", "-c"
}
```

Same fix for post-script (line 300).

### 5. Install Enrollment Enrichment (`cmd/agent/cli/install.go`)

Current state: `doEnrollment()` (line 208) builds a bare `EndpointInfo` with only hostname, OS family, and arch tag. It does NOT call `enrichEndpointInfo()`, so the initial enrollment to the server is missing CPU, memory, OS detail, disk, kernel, and IP addresses. This affects **both Linux and Windows**.

The daemon's `buildEndpointInfo()` in `cmd/agent/main.go:439` does call `enrichEndpointInfo()`, so when the agent daemon starts and re-enrolls, the data gets populated. But the initial enrollment from `install` is thin.

**Fix**: Call `enrichEndpointInfo()` on the endpoint before enrolling in `doEnrollment()`.

```go
// In install.go doEnrollment(), after building EndpointInfo:
endpoint := &pb.EndpointInfo{
    Hostname: hostname,
    OsFamily: mapOsFamily(runtime.GOOS),
    Tags: map[string]string{
        "arch": runtime.GOARCH,
    },
}
enrichEndpointInfo(endpoint)  // <-- add this line
```

`enrichEndpointInfo()` is defined in `cmd/agent/sysinfo_{linux,windows,darwin}.go` with platform-specific build tags — it will resolve correctly on all platforms.

Note: `enrichEndpointInfo` is in package `main` but `doEnrollment` is in package `cli`. Two options:
- **(A)** Move `enrichEndpointInfo` to a shared package (e.g., `internal/agent/sysinfo/`)
- **(B)** Accept the `EndpointInfo` as a parameter in `doEnrollment` and build it in the caller (`RunInstall` in `install.go` calls into `main` package)

Option **(B)** is simpler — `doEnrollment` already accepts `*pb.EndpointInfo` as a parameter via the Enroll call. We just need `RunInstall` or `runInstallHeadless` to call a function that builds a fully-enriched endpoint. The cleanest approach: **export `buildEndpointInfo` from `main` by extracting it to a shared location**, or pass it as a parameter. Since `cli` is a sub-package of `cmd/agent`, the simplest fix is to move `enrichEndpointInfo` + `sysinfo_*.go` files into the `cli` package, or create a small `cmd/agent/sysinfo` package that both `main` and `cli` import.

**Recommended**: Create `cmd/agent/sysinfo/` package with:
- `sysinfo_linux.go` (moved from `cmd/agent/sysinfo_linux.go`)
- `sysinfo_windows.go` (moved from `cmd/agent/sysinfo_windows.go`)
- `sysinfo_darwin.go` (moved from `cmd/agent/sysinfo_darwin.go`)
- `sysinfo.go` — exports `BuildEndpointInfo(logger) *pb.EndpointInfo`

Then both `cmd/agent/main.go` and `cmd/agent/cli/install.go` call `sysinfo.BuildEndpointInfo()`.

### 6. Windows Config Path (`cmd/agent/cli/config.go`)

Current state: `DefaultDataDir()` calls `os.Geteuid()` which doesn't exist on Windows. `defaultConfigPath` is `/etc/patchiq/agent.yaml` (Linux path).

**Fix**: Platform-specific defaults via build tags.

New file `cmd/agent/cli/config_windows.go`:
```go
//go:build windows

package cli

const defaultConfigPath = `C:\ProgramData\PatchIQ\agent.yaml`

func DefaultDataDir() string {
    return `C:\ProgramData\PatchIQ`
}
```

New file `cmd/agent/cli/config_unix.go` (build tag `!windows`):
```go
//go:build !windows

package cli

import (
    "os"
    "path/filepath"
)

const defaultConfigPath = "/etc/patchiq/agent.yaml"

func DefaultDataDir() string {
    if os.Geteuid() == 0 {
        return "/var/lib/patchiq"
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return ".patchiq"
    }
    return filepath.Join(home, ".patchiq")
}
```

Remove `DefaultDataDir()` and `defaultConfigPath` from `config.go` (they move to platform files).

### 7. Agent Downloads UI Fix (`web/src/pages/agent-downloads/AgentDownloadsPage.tsx`)

Current state: Line 44 generates `--server-url` but the agent CLI expects `--server`.

```typescript
// Line 44, current:
return `.\${filename} install --server-url ${serverUrl} --token ${token}`;

// Fix:
return `.\${filename} install --server ${serverUrl} --token ${token}`;
```

Same fix for the Linux command on line 46.

---

## Files Changed

| File | Change | Section |
|------|--------|---------|
| `cmd/agent/sysinfo_windows.go` | Full rewrite — add OS detail, CPU, memory, disk, kernel | §1 |
| `internal/agent/inventory/hardware_windows.go` | Full rewrite — 10 collectors | §2 |
| `internal/agent/inventory/services_windows.go` | Full rewrite — Get-Service implementation | §3 |
| `internal/agent/patcher/shell_windows.go` | New — `scriptShell()` returns powershell | §4 |
| `internal/agent/patcher/shell_unix.go` | New — `scriptShell()` returns sh | §4 |
| `internal/agent/patcher/patcher.go` | Modify lines 168, 300 — use `scriptShell()` | §4 |
| `cmd/agent/sysinfo/sysinfo.go` | New — exports `BuildEndpointInfo()` | §5 |
| `cmd/agent/sysinfo/sysinfo_linux.go` | Moved from `cmd/agent/sysinfo_linux.go` | §5 |
| `cmd/agent/sysinfo/sysinfo_windows.go` | Moved from `cmd/agent/sysinfo_windows.go` | §5 |
| `cmd/agent/sysinfo/sysinfo_darwin.go` | Moved from `cmd/agent/sysinfo_darwin.go` | §5 |
| `cmd/agent/main.go` | Import `sysinfo` package, call `sysinfo.BuildEndpointInfo()` | §5 |
| `cmd/agent/cli/install.go` | Call `sysinfo.BuildEndpointInfo()` in `doEnrollment()` | §5 |
| `cmd/agent/cli/config_windows.go` | New — Windows-specific `DefaultDataDir()` + `defaultConfigPath` | §6 |
| `cmd/agent/cli/config_unix.go` | New — Unix-specific `DefaultDataDir()` + `defaultConfigPath` | §6 |
| `cmd/agent/cli/config.go` | Remove `DefaultDataDir()` + `defaultConfigPath` (moved to platform files) | §6 |
| `web/src/pages/agent-downloads/AgentDownloadsPage.tsx` | Fix `--server-url` → `--server` | §7 |

## Files NOT Changed

- Proto definitions — all existing message types are sufficient
- Server code — already processes all fields correctly
- PM web UI (endpoint detail) — already renders all fields when present
- `metrics_windows.go` — heartbeat metrics already working
- `sysresource_windows.go` — heartbeat resources already working
- Hardware type definitions (`hardware.go`, `software.go`) — no new types needed
- Service management (`service_windows.go`, `service_linux.go`) — already at parity

---

## Testing Strategy

1. **Unit tests** for each collector's JSON parser (mock PowerShell output)
   - `hardware_windows_test.go` — one test per collector function
   - `services_windows_test.go` — parse Get-Service output
   - `sysinfo_windows_test.go` — parse combined enrollment query
2. **Integration test** — start agent on DESKTOP-629B940, verify PM UI shows all fields
3. **Cross-compile check** — `make build-agents` must still succeed (no cgo)

## Performance

- Enrollment: +1 PowerShell call (one-time)
- Hardware collection: +10 PowerShell calls every 24h (~3-5s total)
- Heartbeat: No change (still 3 calls every 60s)
- Services collection: +1 PowerShell call every 24h

## Risks

1. **Elevation**: `Get-Tpm` requires admin rights. If agent runs as service (SYSTEM), this works. If running manually, TPM returns empty. Mitigated by try/catch with warning log.
2. **PowerShell version**: CIM cmdlets require PowerShell 3.0+ (Windows 8.1+). All supported Windows versions have this.
3. **WMI service**: If WMI is disabled, all CIM queries fail. This is an OS configuration issue outside our control. Each collector logs a warning and returns empty data — same as Linux when tools are missing.
