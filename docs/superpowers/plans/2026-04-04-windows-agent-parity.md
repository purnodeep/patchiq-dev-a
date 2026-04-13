# Windows Agent Parity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Achieve full data collection parity between Windows and Linux agents — hardware, services, enrollment, install flow, and script execution.

**Architecture:** Replace all Windows stub collectors with PowerShell/CIM-based implementations that mirror the Linux `/proc`/`lscpu`/`dmidecode` approach. Extract `enrichEndpointInfo` into a shared `cmd/agent/sysinfo` package so both `main` and `cli` can use it. Fix platform-specific config paths and the Agent Downloads UI flag mismatch.

**Tech Stack:** Go 1.25, PowerShell CIM cmdlets (`Get-CimInstance`), `encoding/json` for parsing, `os/exec` for PowerShell invocation. No new dependencies.

**Spec:** `docs/plans/2026-04-04-windows-agent-parity-design.md`

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `cmd/agent/sysinfo/sysinfo.go` | Exports `BuildEndpointInfo(logger) *pb.EndpointInfo` — shared by main + cli |
| `cmd/agent/sysinfo/enrich_linux.go` | Linux `enrichEndpointInfo` (moved from `cmd/agent/sysinfo_linux.go`) |
| `cmd/agent/sysinfo/enrich_windows.go` | Windows `enrichEndpointInfo` (rewrite of `cmd/agent/sysinfo_windows.go`) |
| `cmd/agent/sysinfo/enrich_darwin.go` | macOS `enrichEndpointInfo` (moved from `cmd/agent/sysinfo_darwin.go`) |
| `cmd/agent/sysinfo/parse_windows.go` | Pure parsing for Windows enrollment JSON (no build tag — testable everywhere) |
| `cmd/agent/sysinfo/parse_windows_test.go` | Tests for Windows enrollment parsing |
| `cmd/agent/sysinfo/enrich_linux_test.go` | Tests for Linux enrollment helpers (already tested implicitly, add explicit) |
| `cmd/agent/cli/config_windows.go` | Windows `DefaultDataDir()` + `defaultConfigPath` |
| `cmd/agent/cli/config_unix.go` | Unix `DefaultDataDir()` + `defaultConfigPath` |
| `internal/agent/inventory/hardware_windows_parse.go` | Pure parsing functions for Windows CIM JSON (no build tag — testable everywhere) |
| `internal/agent/inventory/hardware_windows_parse_test.go` | Tests for all 10 CIM parsers |
| `internal/agent/inventory/services_windows_parse.go` | Pure parsing for Get-Service JSON |
| `internal/agent/inventory/services_windows_parse_test.go` | Tests for service parsing |
| `internal/agent/inventory/testdata/windows/` | JSON fixtures for all CIM queries |
| `internal/agent/patcher/shell_windows.go` | `scriptShell()` returns powershell |
| `internal/agent/patcher/shell_unix.go` | `scriptShell()` returns sh (build tag `!windows`) |

### Modified Files

| File | Change |
|------|--------|
| `internal/agent/inventory/hardware_windows.go` | Full rewrite — 10 CIM collector functions |
| `internal/agent/inventory/services_windows.go` | Full rewrite — `Get-Service` implementation |
| `internal/agent/patcher/patcher.go` | Use `scriptShell()` for pre/post scripts |
| `cmd/agent/main.go` | Import `sysinfo` package, replace inline `buildEndpointInfo` |
| `cmd/agent/cli/install.go` | Use `sysinfo.BuildEndpointInfo()` in `doEnrollment` |
| `cmd/agent/cli/config.go` | Remove `DefaultDataDir()` + `defaultConfigPath` |
| `web/src/pages/agent-downloads/AgentDownloadsPage.tsx` | Fix `--server-url` → `--server` |

### Deleted Files

| File | Reason |
|------|--------|
| `cmd/agent/sysinfo_linux.go` | Moved to `cmd/agent/sysinfo/enrich_linux.go` |
| `cmd/agent/sysinfo_windows.go` | Replaced by `cmd/agent/sysinfo/enrich_windows.go` |
| `cmd/agent/sysinfo_darwin.go` | Moved to `cmd/agent/sysinfo/enrich_darwin.go` |

---

## Task 1: Agent Downloads UI Flag Fix (§7)

**Files:**
- Modify: `web/src/pages/agent-downloads/AgentDownloadsPage.tsx:41-46`

- [ ] **Step 1: Fix the flag name in buildInstallCommand**

In `web/src/pages/agent-downloads/AgentDownloadsPage.tsx`, replace `--server-url` with `--server` on both lines:

```typescript
function buildInstallCommand(binary: AgentBinaryInfo, serverUrl: string, token: string): string {
  const filename = binary.filename;
  if (binary.os === 'windows') {
    return `.\\${filename} install --server ${serverUrl} --token ${token}`;
  }
  return `chmod +x ${filename} && sudo ./${filename} install --server ${serverUrl} --token ${token}`;
}
```

- [ ] **Step 2: Verify frontend lint passes**

Run: `cd web && npx eslint src/pages/agent-downloads/AgentDownloadsPage.tsx`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/agent-downloads/AgentDownloadsPage.tsx
git commit -m "fix(web): correct --server-url to --server in agent install command"
```

---

## Task 2: Platform-Specific Config Paths (§6)

**Files:**
- Create: `cmd/agent/cli/config_windows.go`
- Create: `cmd/agent/cli/config_unix.go`
- Modify: `cmd/agent/cli/config.go`
- Modify: `cmd/agent/cli/config_test.go`

- [ ] **Step 1: Create `config_unix.go` with Unix defaults**

Create `cmd/agent/cli/config_unix.go`:

```go
//go:build !windows

package cli

import (
	"os"
	"path/filepath"
)

const defaultConfigPath = "/etc/patchiq/agent.yaml"

// DefaultDataDir returns /var/lib/patchiq if running as root, otherwise ~/.patchiq.
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

- [ ] **Step 2: Create `config_windows.go` with Windows defaults**

Create `cmd/agent/cli/config_windows.go`:

```go
//go:build windows

package cli

import (
	"os"
	"path/filepath"
)

const defaultConfigPath = `C:\ProgramData\PatchIQ\agent.yaml`

// DefaultDataDir returns C:\ProgramData\PatchIQ, or a fallback under the user's
// home directory if ProgramData is not available.
func DefaultDataDir() string {
	if pd := os.Getenv("ProgramData"); pd != "" {
		return filepath.Join(pd, "PatchIQ")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return `C:\PatchIQ`
	}
	return filepath.Join(home, ".patchiq")
}
```

- [ ] **Step 3: Remove `DefaultDataDir` and `defaultConfigPath` from `config.go`**

In `cmd/agent/cli/config.go`, remove the `defaultConfigPath` constant (line 21 area), the `DefaultDataDir()` function (lines 24-34), and the `"os"` and `"path/filepath"` imports that are no longer needed. Keep only `AgentConfig` struct and `LoadAgentConfig()`.

After edit, `config.go` should contain:

```go
package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// AgentConfig holds the configuration for the PatchIQ agent.
type AgentConfig struct {
	ServerAddress string        `koanf:"server_address" yaml:"server_address"`
	DataDir       string        `koanf:"data_dir" yaml:"data_dir"`
	LogLevel      string        `koanf:"log_level" yaml:"log_level"`
	ScanInterval  time.Duration `koanf:"scan_interval" yaml:"scan_interval"`
}

// LoadAgentConfig loads agent configuration with precedence: defaults < file < env vars.
// If configPath is empty or the file does not exist, defaults and env vars are used without error.
func LoadAgentConfig(configPath string) (AgentConfig, error) {
	k := koanf.New(".")

	defaults := map[string]any{
		"server_address": "localhost:50051",
		"data_dir":       DefaultDataDir(),
		"log_level":      "info",
		"scan_interval":  15 * time.Minute,
	}
	for key, val := range defaults {
		k.Set(key, val) //nolint:errcheck
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			if !os.IsNotExist(err) {
				return AgentConfig{}, fmt.Errorf("load agent config: stat %s: %w", configPath, err)
			}
		} else {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return AgentConfig{}, fmt.Errorf("load agent config file %s: %w", configPath, err)
			}
		}
	}

	if err := k.Load(env.Provider("PATCHIQ_AGENT_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "PATCHIQ_AGENT_"))
	}), nil); err != nil {
		return AgentConfig{}, fmt.Errorf("load agent env config: %w", err)
	}

	var cfg AgentConfig
	if err := k.Unmarshal("", &cfg); err != nil {
		return AgentConfig{}, fmt.Errorf("unmarshal agent config: %w", err)
	}

	return cfg, nil
}
```

Note: `os` import is still needed for `os.Stat` in `LoadAgentConfig`. Keep it.

- [ ] **Step 4: Verify build compiles**

Run: `go build ./cmd/agent/...`
Expected: Compiles without errors.

- [ ] **Step 5: Run existing config tests**

Run: `go test ./cmd/agent/cli/ -run TestConfig -v`
Expected: All pass.

- [ ] **Step 6: Commit**

```bash
git add cmd/agent/cli/config.go cmd/agent/cli/config_unix.go cmd/agent/cli/config_windows.go
git commit -m "refactor(agent): platform-specific config paths for Windows/Unix"
```

---

## Task 3: Extract sysinfo Package (§5)

**Files:**
- Create: `cmd/agent/sysinfo/sysinfo.go`
- Create: `cmd/agent/sysinfo/enrich_linux.go` (moved from `cmd/agent/sysinfo_linux.go`)
- Create: `cmd/agent/sysinfo/enrich_windows.go` (moved from `cmd/agent/sysinfo_windows.go`)
- Create: `cmd/agent/sysinfo/enrich_darwin.go` (moved from `cmd/agent/sysinfo_darwin.go`)
- Delete: `cmd/agent/sysinfo_linux.go`, `cmd/agent/sysinfo_windows.go`, `cmd/agent/sysinfo_darwin.go`
- Modify: `cmd/agent/main.go`
- Modify: `cmd/agent/cli/install.go`

- [ ] **Step 1: Create `cmd/agent/sysinfo/sysinfo.go`**

```go
package sysinfo

import (
	"log/slog"
	"os"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

// BuildEndpointInfo constructs an EndpointInfo proto with platform-specific
// hardware and OS details populated. Used by both the daemon and the install CLI.
func BuildEndpointInfo(logger *slog.Logger) *pb.EndpointInfo {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Warn("failed to get hostname for endpoint info", "error", err)
		hostname = "unknown"
	}
	var osFamily pb.OsFamily
	switch runtime.GOOS {
	case "linux":
		osFamily = pb.OsFamily_OS_FAMILY_LINUX
	case "windows":
		osFamily = pb.OsFamily_OS_FAMILY_WINDOWS
	case "darwin":
		osFamily = pb.OsFamily_OS_FAMILY_MACOS
	default:
		logger.Warn("unrecognized OS for endpoint info, reporting as unspecified", "os", runtime.GOOS)
	}
	info := &pb.EndpointInfo{
		Hostname:  hostname,
		OsFamily:  osFamily,
		OsVersion: runtime.GOOS + "/" + runtime.GOARCH,
	}
	EnrichEndpointInfo(info)
	return info
}
```

- [ ] **Step 2: Move Linux sysinfo to `cmd/agent/sysinfo/enrich_linux.go`**

Copy `cmd/agent/sysinfo_linux.go` to `cmd/agent/sysinfo/enrich_linux.go`. Change `package main` to `package sysinfo`. Rename `enrichEndpointInfo` to `EnrichEndpointInfo` (exported).

```go
//go:build linux

package sysinfo

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo) {
	info.OsVersion = readFileField("/etc/os-release", "PRETTY_NAME", runtime.GOOS+"/"+runtime.GOARCH)
	info.CpuType = readCPUModel()
	info.MemoryBytes = readMemTotal()

	if data, err := os.ReadFile("/proc/version"); err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			info.HardwareModel = parts[2]
		}
	}

	addrs := localIPs()
	if len(addrs) > 0 {
		info.IpAddresses = addrs
	}

	if info.Tags == nil {
		info.Tags = make(map[string]string)
	}
	if cores := countCPUCores(); cores > 0 {
		info.Tags["cpu_cores"] = fmt.Sprintf("%d", cores)
	}
	if diskGB := totalDiskGB(); diskGB > 0 {
		info.Tags["disk_total_gb"] = fmt.Sprintf("%d", diskGB)
	}
}

func readFileField(path, key, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), "\"")
		}
	}
	return fallback
}

func readCPUModel() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			if strings.TrimSpace(k) == "model name" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func readMemTotal() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				var kb uint64
				for _, c := range parts[1] {
					if c >= '0' && c <= '9' {
						kb = kb*10 + uint64(c-'0')
					}
				}
				return kb * 1024
			}
		}
	}
	return 0
}

func countCPUCores() int {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "processor") {
			count++
		}
	}
	return count
}

func totalDiskGB() int64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0
	}
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	return int64(totalBytes / (1024 * 1024 * 1024))
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
```

- [ ] **Step 3: Move Darwin sysinfo to `cmd/agent/sysinfo/enrich_darwin.go`**

Copy `cmd/agent/sysinfo_darwin.go` to `cmd/agent/sysinfo/enrich_darwin.go`. Change `package main` → `package sysinfo`, rename `enrichEndpointInfo` → `EnrichEndpointInfo`, `parseSystemProfiler` → `ParseSystemProfiler`, `parseMemoryString` → `ParseMemoryString`.

```go
//go:build darwin

package sysinfo

import (
	"bytes"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo) {
	out, err := exec.Command("system_profiler", "SPHardwareDataType", "SPSoftwareDataType").Output()
	if err != nil {
		return
	}
	ParseSystemProfiler(out, info)
}

func ParseSystemProfiler(data []byte, info *pb.EndpointInfo) {
	for _, line := range bytes.Split(data, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if k, v, ok := strings.Cut(trimmed, ": "); ok {
			switch strings.TrimSpace(k) {
			case "Model Name", "Model Identifier":
				if info.HardwareModel == "" {
					info.HardwareModel = strings.TrimSpace(v)
				}
			case "Chip", "Processor Name":
				if info.CpuType == "" {
					info.CpuType = strings.TrimSpace(v)
				}
			case "Memory":
				info.MemoryBytes = ParseMemoryString(strings.TrimSpace(v))
			case "System Version":
				info.OsVersionDetail = strings.TrimSpace(v)
				info.OsVersion = runtime.GOOS + "/" + runtime.GOARCH
			}
		}
	}
}

func ParseMemoryString(s string) uint64 {
	fields := strings.Fields(s)
	if len(fields) < 2 {
		return 0
	}
	val, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(fields[1]) {
	case "GB":
		return val * 1024 * 1024 * 1024
	case "MB":
		return val * 1024 * 1024
	case "TB":
		return val * 1024 * 1024 * 1024 * 1024
	}
	return 0
}
```

- [ ] **Step 4: Create initial Windows sysinfo stub at `cmd/agent/sysinfo/enrich_windows.go`**

For now, move the existing Windows sysinfo with its current functionality. We'll enhance it in Task 4.

```go
//go:build windows

package sysinfo

import (
	"fmt"
	"net"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo) {
	info.OsVersion = runtime.GOOS + "/" + runtime.GOARCH

	addrs := localIPs()
	if len(addrs) > 0 {
		info.IpAddresses = addrs
	}

	if info.Tags == nil {
		info.Tags = make(map[string]string)
	}
	info.Tags["cpu_cores"] = fmt.Sprintf("%d", runtime.NumCPU())
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
```

- [ ] **Step 5: Delete old sysinfo files from `cmd/agent/`**

```bash
rm cmd/agent/sysinfo_linux.go cmd/agent/sysinfo_windows.go cmd/agent/sysinfo_darwin.go
```

- [ ] **Step 6: Update `cmd/agent/main.go` to use sysinfo package**

In `cmd/agent/main.go`, replace the `buildEndpointInfo` function and its call.

Remove the `buildEndpointInfo` function entirely (lines 439-463).

Add import: `"github.com/skenzeriq/patchiq/cmd/agent/sysinfo"`

Replace all calls to `buildEndpointInfo(logger)` with `sysinfo.BuildEndpointInfo(logger)`.

- [ ] **Step 7: Update `cmd/agent/cli/install.go` to use enriched endpoint info**

In `cmd/agent/cli/install.go`, update `doEnrollment` to use `sysinfo.BuildEndpointInfo`.

Add import: `"github.com/skenzeriq/patchiq/cmd/agent/sysinfo"`

Replace the `doEnrollment` function:

```go
func doEnrollment(ctx context.Context, client comms.Enroller, state *comms.AgentState, token string) (comms.EnrollResult, error) {
	meta := comms.AgentMeta{
		AgentVersion:    "dev",
		ProtocolVersion: 1,
		Capabilities:    []string{"inventory"},
	}

	endpoint := sysinfo.BuildEndpointInfo(slog.Default())

	result, err := comms.Enroll(ctx, client, state, token, meta, endpoint)
	if err != nil {
		return comms.EnrollResult{}, fmt.Errorf("install enrollment: %w", err)
	}
	return result, nil
}
```

Remove the now-unused `os` and `runtime` imports from `install.go` if they become orphaned. Add `"log/slog"` import if not already present.

- [ ] **Step 8: Verify build compiles**

Run: `go build ./cmd/agent/...`
Expected: Compiles without errors.

- [ ] **Step 9: Run existing tests**

Run: `go test ./cmd/agent/... -v -count=1`
Expected: All existing tests pass.

- [ ] **Step 10: Commit**

```bash
git add cmd/agent/sysinfo/ cmd/agent/main.go cmd/agent/cli/install.go
git rm cmd/agent/sysinfo_linux.go cmd/agent/sysinfo_windows.go cmd/agent/sysinfo_darwin.go
git commit -m "refactor(agent): extract sysinfo package for shared enrollment enrichment"
```

---

## Task 4: Windows Enrollment Enrichment (§1)

**Files:**
- Modify: `cmd/agent/sysinfo/enrich_windows.go`
- Create: `cmd/agent/sysinfo/enrich_windows_test.go`

Note: The parse function `parseWinEnrollmentJSON` is defined in the windows-tagged `enrich_windows.go` file. To make it testable on all platforms, we extract it into a separate non-tagged file.

- [ ] **Step 1: Create parse helper `cmd/agent/sysinfo/parse_windows.go` (no build tag)**

```go
package sysinfo

import (
	"encoding/json"
	"strings"
)

// winEnrollmentInfo holds parsed enrollment data from a combined PowerShell query.
type winEnrollmentInfo struct {
	osCaption      string
	osVersion      string
	osBuild        string
	cpuName        string
	memTotalKB     uint64
	diskTotalBytes uint64
}

// parseWinEnrollmentJSON parses the combined JSON output from the enrollment PowerShell query.
func parseWinEnrollmentJSON(raw string) winEnrollmentInfo {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return winEnrollmentInfo{}
	}

	var parsed struct {
		OSCaption      string `json:"os_caption"`
		OSVersion      string `json:"os_version"`
		OSBuild        string `json:"os_build"`
		CPUName        string `json:"cpu_name"`
		MemTotalKB     uint64 `json:"mem_total_kb"`
		DiskTotalBytes uint64 `json:"disk_total_bytes"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return winEnrollmentInfo{}
	}

	return winEnrollmentInfo{
		osCaption:      parsed.OSCaption,
		osVersion:      parsed.OSVersion,
		osBuild:        parsed.OSBuild,
		cpuName:        parsed.CPUName,
		memTotalKB:     parsed.MemTotalKB,
		diskTotalBytes: parsed.DiskTotalBytes,
	}
}
```

- [ ] **Step 2: Write tests for Windows enrollment parsing**

Create `cmd/agent/sysinfo/parse_windows_test.go` (no build tag):

```go
package sysinfo

import (
	"testing"
)

func TestParseWinEnrollmentInfo(t *testing.T) {
	input := `{
		"os_caption": "Microsoft Windows 11 Pro",
		"os_version": "10.0.26100",
		"os_build": "26100",
		"cpu_name": "13th Gen Intel(R) Core(TM) i7-13700K",
		"mem_total_kb": 16777216,
		"disk_total_bytes": 549755813888
	}`

	info := parseWinEnrollmentJSON(input)

	if info.osCaption != "Microsoft Windows 11 Pro" {
		t.Errorf("osCaption = %q, want %q", info.osCaption, "Microsoft Windows 11 Pro")
	}
	if info.osVersion != "10.0.26100" {
		t.Errorf("osVersion = %q, want %q", info.osVersion, "10.0.26100")
	}
	if info.osBuild != "26100" {
		t.Errorf("osBuild = %q, want %q", info.osBuild, "26100")
	}
	if info.cpuName != "13th Gen Intel(R) Core(TM) i7-13700K" {
		t.Errorf("cpuName = %q, want %q", info.cpuName, "13th Gen Intel(R) Core(TM) i7-13700K")
	}
	if info.memTotalKB != 16777216 {
		t.Errorf("memTotalKB = %d, want %d", info.memTotalKB, 16777216)
	}
	if info.diskTotalBytes != 549755813888 {
		t.Errorf("diskTotalBytes = %d, want %d", info.diskTotalBytes, 549755813888)
	}
}

func TestParseWinEnrollmentInfo_Empty(t *testing.T) {
	info := parseWinEnrollmentJSON("")
	if info.osCaption != "" {
		t.Errorf("expected empty osCaption, got %q", info.osCaption)
	}
}

func TestParseWinEnrollmentInfo_Partial(t *testing.T) {
	input := `{"os_caption": "Microsoft Windows 10 Enterprise", "cpu_name": "AMD Ryzen 9 5900X"}`
	info := parseWinEnrollmentJSON(input)
	if info.osCaption != "Microsoft Windows 10 Enterprise" {
		t.Errorf("osCaption = %q", info.osCaption)
	}
	if info.cpuName != "AMD Ryzen 9 5900X" {
		t.Errorf("cpuName = %q", info.cpuName)
	}
	if info.memTotalKB != 0 {
		t.Errorf("memTotalKB should be 0, got %d", info.memTotalKB)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/agent/sysinfo/ -run TestParseWinEnrollment -v`
Expected: FAIL — `parseWinEnrollmentJSON` not defined.

- [ ] **Step 3: Implement Windows enrollment enrichment**

Rewrite `cmd/agent/sysinfo/enrich_windows.go` (the struct and parse function are in `parse_windows.go`):

```go
//go:build windows

package sysinfo

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"runtime"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

func EnrichEndpointInfo(info *pb.EndpointInfo) {
	// Combined CIM query: OS + CPU + Disk in one PowerShell call.
	const psCmd = `$os = Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, BuildNumber, TotalVisibleMemorySize
$cpu = (Get-CimInstance Win32_Processor | Select-Object -First 1).Name
$disk = (Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Measure-Object -Property Size -Sum).Sum
@{
  os_caption = $os.Caption
  os_version = $os.Version
  os_build = $os.BuildNumber
  cpu_name = $cpu
  mem_total_kb = $os.TotalVisibleMemorySize
  disk_total_bytes = $disk
} | ConvertTo-Json -Compress`

	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err == nil {
		ei := parseWinEnrollmentJSON(string(bytes.TrimSpace(out)))
		if ei.osCaption != "" {
			info.OsVersion = ei.osCaption
			info.OsVersionDetail = ei.osCaption + " Build " + ei.osBuild
		}
		if ei.osVersion != "" {
			info.HardwareModel = ei.osVersion
		}
		if ei.cpuName != "" {
			info.CpuType = ei.cpuName
		}
		if ei.memTotalKB > 0 {
			info.MemoryBytes = ei.memTotalKB * 1024
		}
		if ei.diskTotalBytes > 0 {
			if info.Tags == nil {
				info.Tags = make(map[string]string)
			}
			info.Tags["disk_total_gb"] = fmt.Sprintf("%d", ei.diskTotalBytes/(1024*1024*1024))
		}
	}

	// IP addresses.
	addrs := localIPs()
	if len(addrs) > 0 {
		info.IpAddresses = addrs
	}

	// CPU cores and arch.
	if info.Tags == nil {
		info.Tags = make(map[string]string)
	}
	info.Tags["cpu_cores"] = fmt.Sprintf("%d", runtime.NumCPU())
	info.Tags["arch"] = runtime.GOARCH
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var ips []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP.String())
			}
		}
	}
	return ips
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/agent/sysinfo/ -run TestParseWinEnrollment -v`
Expected: All PASS.

- [ ] **Step 5: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: Compiles.

- [ ] **Step 6: Commit**

```bash
git add cmd/agent/sysinfo/enrich_windows.go cmd/agent/sysinfo/enrich_windows_test.go
git commit -m "feat(agent): Windows enrollment enrichment with OS, CPU, memory, disk"
```

---

## Task 5: Windows Hardware Collector — Parse Functions (§2)

**Files:**
- Create: `internal/agent/inventory/hardware_windows_parse.go` (no build tag)
- Create: `internal/agent/inventory/hardware_windows_parse_test.go` (no build tag)
- Create: `internal/agent/inventory/testdata/windows/` (JSON fixtures)

This task builds all the pure parsing functions and tests. No PowerShell calls — just JSON→struct parsing. Testable on any platform.

- [ ] **Step 1: Create testdata fixtures**

Create `internal/agent/inventory/testdata/windows/cpu.json`:
```json
[{"Name":"13th Gen Intel(R) Core(TM) i7-13700K","Manufacturer":"GenuineIntel","Family":198,"NumberOfCores":16,"NumberOfLogicalProcessors":24,"MaxClockSpeed":5400,"L2CacheSize":24576,"L3CacheSize":30720,"Architecture":9,"VirtualizationFirmwareEnabled":true}]
```

Create `internal/agent/inventory/testdata/windows/memory_os.json`:
```json
{"TotalVisibleMemorySize":16777216,"FreePhysicalMemory":8388608}
```

Create `internal/agent/inventory/testdata/windows/memory_dimm.json`:
```json
[{"BankLabel":"BANK 0","DeviceLocator":"DIMM 0","Capacity":17179869184,"SMBIOSMemoryType":34,"ConfiguredClockSpeed":4800,"Manufacturer":"Samsung","SerialNumber":"12345678","PartNumber":"M471A2G43AB2-CWE","FormFactor":12}]
```

Create `internal/agent/inventory/testdata/windows/motherboard.json`:
```json
{"board":{"Manufacturer":"ASUSTeK COMPUTER INC.","Product":"ROG STRIX Z790-E","Version":"Rev 1.xx","SerialNumber":"SN123456"},"bios":{"Manufacturer":"American Megatrends Inc.","SMBIOSBIOSVersion":"2803","ReleaseDate":"2024-01-15T00:00:00Z"}}
```

Create `internal/agent/inventory/testdata/windows/disks.json`:
```json
[{"DeviceID":"\\\\.\\PHYSICALDRIVE0","Model":"Samsung SSD 980 PRO 1TB","SerialNumber":"S6BENS0T123456","Size":1000204886016,"MediaType":"Fixed hard disk media","InterfaceType":"NVMe","FirmwareRevision":"5B2QGXA7","Status":"OK","Partitions":3}]
```

Create `internal/agent/inventory/testdata/windows/logical_disks.json`:
```json
[{"DeviceID":"C:","Size":499971518464,"FreeSpace":249985759232,"FileSystem":"NTFS","VolumeName":"Windows"},{"DeviceID":"D:","Size":500107862016,"FreeSpace":400086089728,"FileSystem":"NTFS","VolumeName":"Data"}]
```

Create `internal/agent/inventory/testdata/windows/gpu.json`:
```json
[{"Name":"NVIDIA GeForce RTX 4090","AdapterRAM":25769803776,"DriverVersion":"32.0.15.6094","PNPDeviceID":"PCI\\VEN_10DE&DEV_2684&SUBSYS_40901043"}]
```

Create `internal/agent/inventory/testdata/windows/network.json`:
```json
{"adapters":[{"Name":"Ethernet","MacAddress":"AA-BB-CC-DD-EE-FF","MtuSize":1500,"Status":"Up","LinkSpeed":"1 Gbps","InterfaceDescription":"Intel(R) Ethernet Controller","DriverName":"e1dexpress"}],"ips":[{"InterfaceAlias":"Ethernet","IPAddress":"192.168.1.50","PrefixLength":24,"AddressFamily":2},{"InterfaceAlias":"Ethernet","IPAddress":"fe80::1234:5678:abcd:ef01","PrefixLength":64,"AddressFamily":23}]}
```

Create `internal/agent/inventory/testdata/windows/usb.json`:
```json
[{"PNPDeviceID":"USB\\VID_046D&PID_C52B\\6&ABC123","Name":"Logitech USB Receiver"},{"PNPDeviceID":"USB\\VID_8087&PID_0029\\0","Name":"Intel(R) Wireless Bluetooth(R)"}]
```

Create `internal/agent/inventory/testdata/windows/battery.json`:
```json
[{"BatteryStatus":2,"EstimatedChargeRemaining":85,"DesignCapacity":50000,"FullChargeCapacity":47500,"Chemistry":2}]
```

Create `internal/agent/inventory/testdata/windows/tpm.json`:
```json
{"TpmPresent":true,"ManufacturerVersion":"2.0"}
```

Create `internal/agent/inventory/testdata/windows/computer_system.json`:
```json
{"Model":"ASUS ROG Strix","HypervisorPresent":false}
```

Create `internal/agent/inventory/testdata/windows/computer_system_vm.json`:
```json
{"Model":"Virtual Machine","HypervisorPresent":true}
```

Create `internal/agent/inventory/testdata/windows/services.json`:
```json
[{"Name":"wuauserv","DisplayName":"Windows Update","Status":4,"StartType":3},{"Name":"WinDefend","DisplayName":"Microsoft Defender Antivirus Service","Status":4,"StartType":2},{"Name":"Spooler","DisplayName":"Print Spooler","Status":1,"StartType":3},{"Name":"MSSQLSERVER","DisplayName":"SQL Server (MSSQLSERVER)","Status":4,"StartType":2}]
```

- [ ] **Step 2: Write parse functions and tests for CPU**

Create `internal/agent/inventory/hardware_windows_parse.go`:

```go
package inventory

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// winCPUEntry mirrors Win32_Processor CIM fields.
type winCPUEntry struct {
	Name                        string `json:"Name"`
	Manufacturer                string `json:"Manufacturer"`
	Family                      int    `json:"Family"`
	NumberOfCores               int    `json:"NumberOfCores"`
	NumberOfLogicalProcessors   int    `json:"NumberOfLogicalProcessors"`
	MaxClockSpeed               int    `json:"MaxClockSpeed"`
	L2CacheSize                 int    `json:"L2CacheSize"`
	L3CacheSize                 int    `json:"L3CacheSize"`
	Architecture                int    `json:"Architecture"`
	VirtualizationFirmwareEnabled bool `json:"VirtualizationFirmwareEnabled"`
}

func parseWinCPU(data string) CPUInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return CPUInfo{}
	}

	var entries []winCPUEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return CPUInfo{}
		}
	} else {
		var single winCPUEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return CPUInfo{}
		}
		entries = []winCPUEntry{single}
	}

	if len(entries) == 0 {
		return CPUInfo{}
	}

	first := entries[0]
	info := CPUInfo{
		ModelName:      first.Name,
		Vendor:         first.Manufacturer,
		CoresPerSocket: first.NumberOfCores,
		TotalLogical:   first.NumberOfLogicalProcessors,
		Sockets:        len(entries),
		MaxMHz:         float64(first.MaxClockSpeed),
		Architecture:   winArchString(first.Architecture),
	}

	if first.NumberOfCores > 0 {
		info.ThreadsPerCore = first.NumberOfLogicalProcessors / first.NumberOfCores
	}
	if first.L2CacheSize > 0 {
		info.CacheL2 = fmt.Sprintf("%d KiB", first.L2CacheSize)
	}
	if first.L3CacheSize > 0 {
		info.CacheL3 = fmt.Sprintf("%d KiB", first.L3CacheSize)
	}
	if first.VirtualizationFirmwareEnabled {
		info.VirtType = "hyper-v"
	}

	return info
}

func winArchString(arch int) string {
	switch arch {
	case 0:
		return "x86"
	case 9:
		return "x86_64"
	case 12:
		return "ARM64"
	default:
		return strconv.Itoa(arch)
	}
}
```

Create `internal/agent/inventory/hardware_windows_parse_test.go`:

```go
package inventory

import (
	"os"
	"testing"
)

func TestParseWinCPU(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/cpu.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	cpu := parseWinCPU(string(data))

	if cpu.ModelName != "13th Gen Intel(R) Core(TM) i7-13700K" {
		t.Errorf("ModelName = %q", cpu.ModelName)
	}
	if cpu.Vendor != "GenuineIntel" {
		t.Errorf("Vendor = %q", cpu.Vendor)
	}
	if cpu.CoresPerSocket != 16 {
		t.Errorf("CoresPerSocket = %d, want 16", cpu.CoresPerSocket)
	}
	if cpu.TotalLogical != 24 {
		t.Errorf("TotalLogical = %d, want 24", cpu.TotalLogical)
	}
	if cpu.ThreadsPerCore != 1 {
		t.Errorf("ThreadsPerCore = %d, want 1", cpu.ThreadsPerCore)
	}
	if cpu.MaxMHz != 5400 {
		t.Errorf("MaxMHz = %f, want 5400", cpu.MaxMHz)
	}
	if cpu.Architecture != "x86_64" {
		t.Errorf("Architecture = %q, want x86_64", cpu.Architecture)
	}
	if cpu.CacheL2 != "24576 KiB" {
		t.Errorf("CacheL2 = %q", cpu.CacheL2)
	}
	if cpu.CacheL3 != "30720 KiB" {
		t.Errorf("CacheL3 = %q", cpu.CacheL3)
	}
	if cpu.VirtType != "hyper-v" {
		t.Errorf("VirtType = %q, want hyper-v", cpu.VirtType)
	}
	if cpu.Sockets != 1 {
		t.Errorf("Sockets = %d, want 1", cpu.Sockets)
	}
}

func TestParseWinCPU_Empty(t *testing.T) {
	cpu := parseWinCPU("")
	if cpu.ModelName != "" {
		t.Errorf("expected empty, got %q", cpu.ModelName)
	}
}
```

- [ ] **Step 3: Run CPU parse test**

Run: `go test ./internal/agent/inventory/ -run TestParseWinCPU -v`
Expected: PASS.

- [ ] **Step 4: Add remaining parse functions to `hardware_windows_parse.go`**

Append these to `internal/agent/inventory/hardware_windows_parse.go`:

```go
// --- Memory ---

type winMemOSInfo struct {
	TotalVisibleMemorySize uint64 `json:"TotalVisibleMemorySize"`
	FreePhysicalMemory     uint64 `json:"FreePhysicalMemory"`
}

type winDIMMEntry struct {
	BankLabel            string `json:"BankLabel"`
	DeviceLocator        string `json:"DeviceLocator"`
	Capacity             uint64 `json:"Capacity"`
	SMBIOSMemoryType     int    `json:"SMBIOSMemoryType"`
	ConfiguredClockSpeed int    `json:"ConfiguredClockSpeed"`
	Manufacturer         string `json:"Manufacturer"`
	SerialNumber         string `json:"SerialNumber"`
	PartNumber           string `json:"PartNumber"`
	FormFactor           int    `json:"FormFactor"`
}

func parseWinMemory(osData, dimmData string) MemoryInfo {
	info := MemoryInfo{}

	osData = strings.TrimSpace(osData)
	if osData != "" {
		var osInfo winMemOSInfo
		if err := json.Unmarshal([]byte(osData), &osInfo); err == nil {
			info.TotalBytes = osInfo.TotalVisibleMemorySize * 1024
			info.AvailableBytes = osInfo.FreePhysicalMemory * 1024
		}
	}

	dimmData = strings.TrimSpace(dimmData)
	if dimmData == "" {
		return info
	}

	var dimms []winDIMMEntry
	if strings.HasPrefix(dimmData, "[") {
		if err := json.Unmarshal([]byte(dimmData), &dimms); err != nil {
			return info
		}
	} else {
		var single winDIMMEntry
		if err := json.Unmarshal([]byte(dimmData), &single); err != nil {
			return info
		}
		dimms = []winDIMMEntry{single}
	}

	info.NumSlots = len(dimms)
	for _, d := range dimms {
		dimm := DIMMInfo{
			Locator:      d.DeviceLocator,
			BankLocator:  d.BankLabel,
			SizeMB:       int(d.Capacity / (1024 * 1024)),
			SpeedMHz:     d.ConfiguredClockSpeed,
			Manufacturer: d.Manufacturer,
			SerialNumber: d.SerialNumber,
			PartNumber:   strings.TrimSpace(d.PartNumber),
			Type:         winMemTypeString(d.SMBIOSMemoryType),
			FormFactor:   winFormFactorString(d.FormFactor),
		}
		info.DIMMs = append(info.DIMMs, dimm)
	}

	return info
}

func winMemTypeString(t int) string {
	switch t {
	case 20:
		return "DDR"
	case 21:
		return "DDR2"
	case 24:
		return "DDR3"
	case 26:
		return "DDR4"
	case 34:
		return "DDR5"
	default:
		return "Unknown"
	}
}

func winFormFactorString(f int) string {
	switch f {
	case 8:
		return "DIMM"
	case 12:
		return "SODIMM"
	default:
		return "Unknown"
	}
}

// --- Motherboard ---

type winMotherboardJSON struct {
	Board struct {
		Manufacturer string `json:"Manufacturer"`
		Product      string `json:"Product"`
		Version      string `json:"Version"`
		SerialNumber string `json:"SerialNumber"`
	} `json:"board"`
	BIOS struct {
		Manufacturer       string `json:"Manufacturer"`
		SMBIOSBIOSVersion  string `json:"SMBIOSBIOSVersion"`
		ReleaseDate        string `json:"ReleaseDate"`
	} `json:"bios"`
}

func parseWinMotherboard(data string) MotherboardInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return MotherboardInfo{}
	}
	var raw winMotherboardJSON
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return MotherboardInfo{}
	}
	releaseDate := raw.BIOS.ReleaseDate
	if idx := strings.Index(releaseDate, "T"); idx > 0 {
		releaseDate = releaseDate[:idx]
	}
	return MotherboardInfo{
		BoardManufacturer: raw.Board.Manufacturer,
		BoardProduct:      raw.Board.Product,
		BoardVersion:      raw.Board.Version,
		BoardSerial:       raw.Board.SerialNumber,
		BIOSVendor:        raw.BIOS.Manufacturer,
		BIOSVersion:       raw.BIOS.SMBIOSBIOSVersion,
		BIOSReleaseDate:   releaseDate,
	}
}

// --- Storage ---

type winDiskDriveEntry struct {
	DeviceID         string `json:"DeviceID"`
	Model            string `json:"Model"`
	SerialNumber     string `json:"SerialNumber"`
	Size             uint64 `json:"Size"`
	MediaType        string `json:"MediaType"`
	InterfaceType    string `json:"InterfaceType"`
	FirmwareRevision string `json:"FirmwareRevision"`
	Status           string `json:"Status"`
}

type winLogicalDiskEntry struct {
	DeviceID   string `json:"DeviceID"`
	Size       uint64 `json:"Size"`
	FreeSpace  uint64 `json:"FreeSpace"`
	FileSystem string `json:"FileSystem"`
	VolumeName string `json:"VolumeName"`
}

func parseWinStorage(diskData, logicalData string) []StorageDevice {
	var drives []winDiskDriveEntry
	diskData = strings.TrimSpace(diskData)
	if diskData != "" {
		if strings.HasPrefix(diskData, "[") {
			json.Unmarshal([]byte(diskData), &drives) //nolint:errcheck
		} else {
			var single winDiskDriveEntry
			if err := json.Unmarshal([]byte(diskData), &single); err == nil {
				drives = []winDiskDriveEntry{single}
			}
		}
	}

	var logicals []winLogicalDiskEntry
	logicalData = strings.TrimSpace(logicalData)
	if logicalData != "" {
		if strings.HasPrefix(logicalData, "[") {
			json.Unmarshal([]byte(logicalData), &logicals) //nolint:errcheck
		} else {
			var single winLogicalDiskEntry
			if err := json.Unmarshal([]byte(logicalData), &single); err == nil {
				logicals = []winLogicalDiskEntry{single}
			}
		}
	}

	var devices []StorageDevice
	for _, d := range drives {
		smartStatus := "N/A"
		if d.Status == "OK" {
			smartStatus = "PASSED"
		}
		dev := StorageDevice{
			Name:            d.DeviceID,
			Model:           d.Model,
			Serial:          d.SerialNumber,
			SizeBytes:       d.Size,
			Type:            winDiskType(d.MediaType, d.InterfaceType),
			FirmwareVersion: d.FirmwareRevision,
			Transport:       d.InterfaceType,
			SmartStatus:     smartStatus,
		}

		for _, l := range logicals {
			var usePct int
			if l.Size > 0 {
				usePct = int(float64(l.Size-l.FreeSpace) / float64(l.Size) * 100)
			}
			dev.Partitions = append(dev.Partitions, PartitionInfo{
				Name:       l.DeviceID,
				SizeBytes:  l.Size,
				FSType:     l.FileSystem,
				MountPoint: l.DeviceID,
				UsagePct:   usePct,
			})
		}

		devices = append(devices, dev)
	}

	return devices
}

func winDiskType(mediaType, interfaceType string) string {
	if strings.EqualFold(interfaceType, "NVMe") {
		return "nvme"
	}
	mt := strings.ToLower(mediaType)
	if strings.Contains(mt, "solid state") || strings.Contains(mt, "ssd") {
		return "ssd"
	}
	if strings.Contains(mt, "fixed hard disk") {
		return "hdd"
	}
	return "unknown"
}

// --- GPU ---

type winGPUEntry struct {
	Name          string `json:"Name"`
	AdapterRAM    uint64 `json:"AdapterRAM"`
	DriverVersion string `json:"DriverVersion"`
	PNPDeviceID   string `json:"PNPDeviceID"`
}

func parseWinGPU(data string) []GPUInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	var entries []winGPUEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return nil
		}
	} else {
		var single winGPUEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return nil
		}
		entries = []winGPUEntry{single}
	}

	var gpus []GPUInfo
	for _, e := range entries {
		pciSlot := ""
		if idx := strings.Index(e.PNPDeviceID, "PCI\\"); idx >= 0 {
			pciSlot = e.PNPDeviceID[idx:]
		}
		gpus = append(gpus, GPUInfo{
			Model:         e.Name,
			VRAMMB:        int(e.AdapterRAM / (1024 * 1024)),
			DriverVersion: e.DriverVersion,
			PCISlot:       pciSlot,
		})
	}
	return gpus
}

// --- Network ---

type winNetworkJSON struct {
	Adapters []winNetAdapter    `json:"adapters"`
	IPs      []winNetIPAddress  `json:"ips"`
}

type winNetAdapter struct {
	Name                 string `json:"Name"`
	MacAddress           string `json:"MacAddress"`
	MtuSize              int    `json:"MtuSize"`
	Status               string `json:"Status"`
	LinkSpeed            string `json:"LinkSpeed"`
	InterfaceDescription string `json:"InterfaceDescription"`
	DriverName           string `json:"DriverName"`
}

type winNetIPAddress struct {
	InterfaceAlias string `json:"InterfaceAlias"`
	IPAddress      string `json:"IPAddress"`
	PrefixLength   int    `json:"PrefixLength"`
	AddressFamily  int    `json:"AddressFamily"`
}

func parseWinNetwork(data string) []NetworkInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	var raw winNetworkJSON
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil
	}

	var nics []NetworkInfo
	for _, a := range raw.Adapters {
		nic := NetworkInfo{
			Name:       a.Name,
			MACAddress: strings.ReplaceAll(a.MacAddress, "-", ":"),
			MTU:        a.MtuSize,
			State:      strings.ToLower(a.Status),
			SpeedMbps:  parseWinLinkSpeed(a.LinkSpeed),
			Type:       classifyWinNetType(a.InterfaceDescription),
			Driver:     a.DriverName,
		}

		for _, ip := range raw.IPs {
			if ip.InterfaceAlias != a.Name {
				continue
			}
			addr := IPAddress{Address: ip.IPAddress, PrefixLen: ip.PrefixLength}
			if ip.AddressFamily == 2 {
				nic.IPv4Addresses = append(nic.IPv4Addresses, addr)
			} else if ip.AddressFamily == 23 {
				nic.IPv6Addresses = append(nic.IPv6Addresses, addr)
			}
		}

		nics = append(nics, nic)
	}
	return nics
}

func parseWinLinkSpeed(s string) int {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	val, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToLower(parts[1])
	switch unit {
	case "gbps":
		return int(val * 1000)
	case "mbps":
		return int(val)
	case "kbps":
		return int(val / 1000)
	}
	return 0
}

func classifyWinNetType(desc string) string {
	lower := strings.ToLower(desc)
	switch {
	case strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wireless") || strings.Contains(lower, "wlan"):
		return "wifi"
	case strings.Contains(lower, "bluetooth"):
		return "other"
	case strings.Contains(lower, "virtual") || strings.Contains(lower, "hyper-v") || strings.Contains(lower, "vmware"):
		return "virtual"
	default:
		return "ethernet"
	}
}

// --- USB ---

type winPnPEntry struct {
	PNPDeviceID string `json:"PNPDeviceID"`
	Name        string `json:"Name"`
}

func parseWinUSB(data string) []USBDevice {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	var entries []winPnPEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return nil
		}
	} else {
		var single winPnPEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return nil
		}
		entries = []winPnPEntry{single}
	}

	var devices []USBDevice
	for _, e := range entries {
		vid, pid := extractUSBIDs(e.PNPDeviceID)
		devices = append(devices, USBDevice{
			Bus:         "USB",
			VendorID:    vid,
			ProductID:   pid,
			Description: e.Name,
		})
	}
	return devices
}

func extractUSBIDs(pnpID string) (vendorID, productID string) {
	upper := strings.ToUpper(pnpID)
	if idx := strings.Index(upper, "VID_"); idx >= 0 {
		end := idx + 4 + 4
		if end <= len(pnpID) {
			vendorID = strings.ToLower(pnpID[idx+4 : end])
		}
	}
	if idx := strings.Index(upper, "PID_"); idx >= 0 {
		end := idx + 4 + 4
		if end <= len(pnpID) {
			productID = strings.ToLower(pnpID[idx+4 : end])
		}
	}
	return
}

// --- Battery ---

type winBatteryEntry struct {
	BatteryStatus             int `json:"BatteryStatus"`
	EstimatedChargeRemaining  int `json:"EstimatedChargeRemaining"`
	DesignCapacity            int `json:"DesignCapacity"`
	FullChargeCapacity        int `json:"FullChargeCapacity"`
	Chemistry                 int `json:"Chemistry"`
}

func parseWinBattery(data string) BatteryInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return BatteryInfo{Present: false}
	}

	var entries []winBatteryEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return BatteryInfo{Present: false}
		}
	} else {
		var single winBatteryEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return BatteryInfo{Present: false}
		}
		entries = []winBatteryEntry{single}
	}

	if len(entries) == 0 {
		return BatteryInfo{Present: false}
	}

	b := entries[0]
	info := BatteryInfo{
		Present:     true,
		CapacityPct: b.EstimatedChargeRemaining,
		Status:      winBatteryStatusString(b.BatteryStatus),
		Technology:  winChemistryString(b.Chemistry),
	}

	if b.FullChargeCapacity > 0 {
		info.EnergyFullWh = float64(b.FullChargeCapacity) / 1000.0
	}
	if b.DesignCapacity > 0 {
		info.EnergyDesignWh = float64(b.DesignCapacity) / 1000.0
	}
	if info.EnergyDesignWh > 0 && info.EnergyFullWh > 0 {
		info.HealthPct = int(info.EnergyFullWh / info.EnergyDesignWh * 100)
	}

	return info
}

func winBatteryStatusString(s int) string {
	switch s {
	case 1:
		return "Discharging"
	case 2:
		return "AC"
	case 3:
		return "Full"
	case 4:
		return "Low"
	case 5:
		return "Critical"
	default:
		return "Unknown"
	}
}

func winChemistryString(c int) string {
	switch c {
	case 1:
		return "Other"
	case 2:
		return "Unknown"
	case 3:
		return "Lead Acid"
	case 4:
		return "NiCd"
	case 5:
		return "NiMH"
	case 6:
		return "Li-ion"
	case 7:
		return "Zinc Air"
	case 8:
		return "LiPo"
	default:
		return "Unknown"
	}
}

// --- TPM ---

type winTPMInfo struct {
	TpmPresent          bool   `json:"TpmPresent"`
	ManufacturerVersion string `json:"ManufacturerVersion"`
}

func parseWinTPM(data string) TPMInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return TPMInfo{}
	}
	var raw winTPMInfo
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return TPMInfo{}
	}
	return TPMInfo{
		Present: raw.TpmPresent,
		Version: raw.ManufacturerVersion,
	}
}

// --- Virtualization ---

type winComputerSystem struct {
	Model             string `json:"Model"`
	HypervisorPresent bool   `json:"HypervisorPresent"`
}

func parseWinVirtualization(data string) VirtInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return VirtInfo{}
	}
	var raw winComputerSystem
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return VirtInfo{}
	}

	if !raw.HypervisorPresent && !strings.Contains(strings.ToLower(raw.Model), "virtual") {
		return VirtInfo{}
	}

	hypervisor := "unknown"
	model := strings.ToLower(raw.Model)
	switch {
	case strings.Contains(model, "vmware"):
		hypervisor = "vmware"
	case strings.Contains(model, "virtualbox"):
		hypervisor = "virtualbox"
	case strings.Contains(model, "hyper-v") || strings.Contains(model, "virtual machine"):
		hypervisor = "hyper-v"
	case strings.Contains(model, "kvm") || strings.Contains(model, "qemu"):
		hypervisor = "kvm"
	}

	return VirtInfo{
		IsVirtual:      true,
		HypervisorType: hypervisor,
	}
}
```

- [ ] **Step 5: Add remaining tests to `hardware_windows_parse_test.go`**

Append to `internal/agent/inventory/hardware_windows_parse_test.go`:

```go
func TestParseWinMemory(t *testing.T) {
	osData, err := os.ReadFile("testdata/windows/memory_os.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	dimmData, err := os.ReadFile("testdata/windows/memory_dimm.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	mem := parseWinMemory(string(osData), string(dimmData))

	if mem.TotalBytes != 16777216*1024 {
		t.Errorf("TotalBytes = %d", mem.TotalBytes)
	}
	if mem.AvailableBytes != 8388608*1024 {
		t.Errorf("AvailableBytes = %d", mem.AvailableBytes)
	}
	if mem.NumSlots != 1 {
		t.Errorf("NumSlots = %d", mem.NumSlots)
	}
	if len(mem.DIMMs) != 1 {
		t.Fatalf("expected 1 DIMM, got %d", len(mem.DIMMs))
	}
	d := mem.DIMMs[0]
	if d.SizeMB != 16384 {
		t.Errorf("DIMM SizeMB = %d, want 16384", d.SizeMB)
	}
	if d.Type != "DDR5" {
		t.Errorf("DIMM Type = %q, want DDR5", d.Type)
	}
	if d.SpeedMHz != 4800 {
		t.Errorf("DIMM SpeedMHz = %d", d.SpeedMHz)
	}
	if d.FormFactor != "SODIMM" {
		t.Errorf("DIMM FormFactor = %q", d.FormFactor)
	}
}

func TestParseWinMotherboard(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/motherboard.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	mb := parseWinMotherboard(string(data))

	if mb.BoardManufacturer != "ASUSTeK COMPUTER INC." {
		t.Errorf("BoardManufacturer = %q", mb.BoardManufacturer)
	}
	if mb.BoardProduct != "ROG STRIX Z790-E" {
		t.Errorf("BoardProduct = %q", mb.BoardProduct)
	}
	if mb.BIOSVendor != "American Megatrends Inc." {
		t.Errorf("BIOSVendor = %q", mb.BIOSVendor)
	}
	if mb.BIOSVersion != "2803" {
		t.Errorf("BIOSVersion = %q", mb.BIOSVersion)
	}
	if mb.BIOSReleaseDate != "2024-01-15" {
		t.Errorf("BIOSReleaseDate = %q", mb.BIOSReleaseDate)
	}
}

func TestParseWinStorage(t *testing.T) {
	diskData, err := os.ReadFile("testdata/windows/disks.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	logicalData, err := os.ReadFile("testdata/windows/logical_disks.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	devices := parseWinStorage(string(diskData), string(logicalData))
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	d := devices[0]
	if d.Model != "Samsung SSD 980 PRO 1TB" {
		t.Errorf("Model = %q", d.Model)
	}
	if d.Type != "nvme" {
		t.Errorf("Type = %q, want nvme", d.Type)
	}
	if d.SmartStatus != "PASSED" {
		t.Errorf("SmartStatus = %q", d.SmartStatus)
	}
	if len(d.Partitions) != 2 {
		t.Fatalf("expected 2 partitions, got %d", len(d.Partitions))
	}
	if d.Partitions[0].Name != "C:" {
		t.Errorf("partition[0].Name = %q", d.Partitions[0].Name)
	}
	if d.Partitions[0].FSType != "NTFS" {
		t.Errorf("partition[0].FSType = %q", d.Partitions[0].FSType)
	}
}

func TestParseWinGPU(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/gpu.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	gpus := parseWinGPU(string(data))
	if len(gpus) != 1 {
		t.Fatalf("expected 1 GPU, got %d", len(gpus))
	}
	if gpus[0].Model != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Model = %q", gpus[0].Model)
	}
	if gpus[0].VRAMMB != 24576 {
		t.Errorf("VRAMMB = %d, want 24576", gpus[0].VRAMMB)
	}
}

func TestParseWinNetwork(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/network.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	nics := parseWinNetwork(string(data))
	if len(nics) != 1 {
		t.Fatalf("expected 1 NIC, got %d", len(nics))
	}
	n := nics[0]
	if n.Name != "Ethernet" {
		t.Errorf("Name = %q", n.Name)
	}
	if n.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %q, want AA:BB:CC:DD:EE:FF", n.MACAddress)
	}
	if n.SpeedMbps != 1000 {
		t.Errorf("SpeedMbps = %d, want 1000", n.SpeedMbps)
	}
	if len(n.IPv4Addresses) != 1 {
		t.Fatalf("expected 1 IPv4, got %d", len(n.IPv4Addresses))
	}
	if n.IPv4Addresses[0].Address != "192.168.1.50" {
		t.Errorf("IPv4 = %q", n.IPv4Addresses[0].Address)
	}
	if len(n.IPv6Addresses) != 1 {
		t.Fatalf("expected 1 IPv6, got %d", len(n.IPv6Addresses))
	}
}

func TestParseWinUSB(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/usb.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	devices := parseWinUSB(string(data))
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].VendorID != "046d" {
		t.Errorf("VendorID = %q, want 046d", devices[0].VendorID)
	}
	if devices[0].ProductID != "c52b" {
		t.Errorf("ProductID = %q, want c52b", devices[0].ProductID)
	}
	if devices[0].Description != "Logitech USB Receiver" {
		t.Errorf("Description = %q", devices[0].Description)
	}
}

func TestParseWinBattery(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/battery.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	bat := parseWinBattery(string(data))
	if !bat.Present {
		t.Error("expected Present=true")
	}
	if bat.CapacityPct != 85 {
		t.Errorf("CapacityPct = %d, want 85", bat.CapacityPct)
	}
	if bat.HealthPct != 95 {
		t.Errorf("HealthPct = %d, want 95", bat.HealthPct)
	}
	if bat.Status != "AC" {
		t.Errorf("Status = %q, want AC", bat.Status)
	}
	if bat.Technology != "Unknown" {
		t.Errorf("Technology = %q, want Unknown", bat.Technology)
	}
}

func TestParseWinBattery_Empty(t *testing.T) {
	bat := parseWinBattery("")
	if bat.Present {
		t.Error("expected Present=false for empty input")
	}
}

func TestParseWinTPM(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/tpm.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	tpm := parseWinTPM(string(data))
	if !tpm.Present {
		t.Error("expected Present=true")
	}
	if tpm.Version != "2.0" {
		t.Errorf("Version = %q, want 2.0", tpm.Version)
	}
}

func TestParseWinVirtualization_Bare(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/computer_system.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	virt := parseWinVirtualization(string(data))
	if virt.IsVirtual {
		t.Error("expected IsVirtual=false for bare metal")
	}
}

func TestParseWinVirtualization_VM(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/computer_system_vm.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	virt := parseWinVirtualization(string(data))
	if !virt.IsVirtual {
		t.Error("expected IsVirtual=true for VM")
	}
	if virt.HypervisorType != "hyper-v" {
		t.Errorf("HypervisorType = %q, want hyper-v", virt.HypervisorType)
	}
}

func TestExtractUSBIDs(t *testing.T) {
	tests := []struct {
		pnpID   string
		wantVID string
		wantPID string
	}{
		{"USB\\VID_046D&PID_C52B\\6&ABC123", "046d", "c52b"},
		{"USB\\VID_8087&PID_0029\\0", "8087", "0029"},
		{"PCI\\VEN_10DE", "", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		vid, pid := extractUSBIDs(tt.pnpID)
		if vid != tt.wantVID || pid != tt.wantPID {
			t.Errorf("extractUSBIDs(%q) = (%q, %q), want (%q, %q)", tt.pnpID, vid, pid, tt.wantVID, tt.wantPID)
		}
	}
}

func TestParseWinLinkSpeed(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1 Gbps", 1000},
		{"100 Mbps", 100},
		{"2.5 Gbps", 2500},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		got := parseWinLinkSpeed(tt.input)
		if got != tt.want {
			t.Errorf("parseWinLinkSpeed(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
```

- [ ] **Step 6: Run all parse tests**

Run: `go test ./internal/agent/inventory/ -run "TestParseWin|TestExtractUSB|TestParseWinLinkSpeed" -v`
Expected: All PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/agent/inventory/hardware_windows_parse.go internal/agent/inventory/hardware_windows_parse_test.go internal/agent/inventory/testdata/windows/
git commit -m "feat(agent): Windows hardware CIM parse functions with tests"
```

---

## Task 6: Windows Hardware Collector — CIM Execution (§2)

**Files:**
- Modify: `internal/agent/inventory/hardware_windows.go`

- [ ] **Step 1: Implement the full hardware collector**

Rewrite `internal/agent/inventory/hardware_windows.go`:

```go
//go:build windows

package inventory

import (
	"bytes"
	"context"
	"log/slog"
	"os/exec"
)

// CollectHardware gathers deep hardware inventory from a Windows endpoint
// using PowerShell CIM queries. Each subsystem failure is logged as a warning
// but does not fail the overall collection — partial data is returned.
func CollectHardware(ctx context.Context, logger *slog.Logger) (*HardwareInfo, error) {
	hw := &HardwareInfo{}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_Processor | Select-Object Name, Manufacturer, Family, NumberOfCores, NumberOfLogicalProcessors, MaxClockSpeed, L2CacheSize, L3CacheSize, Architecture, VirtualizationFirmwareEnabled | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: cpu query failed", "error", err)
	} else {
		hw.CPU = parseWinCPU(out)
	}

	memOS := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: memory OS query failed", "error", err)
	} else {
		memOS = out
	}
	memDIMM := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_PhysicalMemory | Select-Object BankLabel, DeviceLocator, Capacity, SMBIOSMemoryType, ConfiguredClockSpeed, Manufacturer, SerialNumber, PartNumber, FormFactor | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: memory DIMM query failed", "error", err)
	} else {
		memDIMM = out
	}
	hw.Memory = parseWinMemory(memOS, memDIMM)

	if out, err := runPSCtx(ctx, `$b = Get-CimInstance Win32_BaseBoard | Select-Object Manufacturer, Product, Version, SerialNumber; $i = Get-CimInstance Win32_BIOS | Select-Object Manufacturer, SMBIOSBIOSVersion, ReleaseDate; @{board=$b; bios=$i} | ConvertTo-Json -Compress -Depth 3`); err != nil {
		logger.Warn("hardware collector: motherboard query failed", "error", err)
	} else {
		hw.Motherboard = parseWinMotherboard(out)
	}

	diskRaw := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_DiskDrive | Select-Object DeviceID, Model, SerialNumber, Size, MediaType, InterfaceType, FirmwareRevision, Status, Partitions | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: disk drive query failed", "error", err)
	} else {
		diskRaw = out
	}
	logicalRaw := ""
	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_LogicalDisk -Filter 'DriveType=3' | Select-Object DeviceID, Size, FreeSpace, FileSystem, VolumeName | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: logical disk query failed", "error", err)
	} else {
		logicalRaw = out
	}
	hw.Storage = parseWinStorage(diskRaw, logicalRaw)

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_VideoController | Select-Object Name, AdapterRAM, DriverVersion, PNPDeviceID | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: gpu query failed", "error", err)
	} else {
		hw.GPU = parseWinGPU(out)
	}

	if out, err := runPSCtx(ctx, `$a = Get-NetAdapter | Where-Object { $_.Status -eq 'Up' -or $_.Status -eq 'Disconnected' } | Select-Object Name, MacAddress, MtuSize, Status, LinkSpeed, InterfaceDescription, DriverName; $i = Get-NetIPAddress -ErrorAction SilentlyContinue | Select-Object InterfaceAlias, IPAddress, PrefixLength, AddressFamily; @{adapters=$a; ips=$i} | ConvertTo-Json -Compress -Depth 3`); err != nil {
		logger.Warn("hardware collector: network query failed", "error", err)
	} else {
		hw.Network = parseWinNetwork(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_PnPEntity | Where-Object { $_.PNPDeviceID -like 'USB\*' } | Select-Object PNPDeviceID, Name | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: usb query failed", "error", err)
	} else {
		hw.USB = parseWinUSB(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_Battery | Select-Object BatteryStatus, EstimatedChargeRemaining, DesignCapacity, FullChargeCapacity, Chemistry | ConvertTo-Json -Compress`); err != nil {
		logger.Debug("hardware collector: battery query failed (expected on desktops)", "error", err)
	} else {
		hw.Battery = parseWinBattery(out)
	}

	if out, err := runPSCtx(ctx, `Get-Tpm -ErrorAction SilentlyContinue | Select-Object TpmPresent, ManufacturerVersion | ConvertTo-Json -Compress`); err != nil {
		logger.Debug("hardware collector: tpm query failed (may need elevation)", "error", err)
	} else {
		hw.TPM = parseWinTPM(out)
	}

	if out, err := runPSCtx(ctx, `Get-CimInstance Win32_ComputerSystem | Select-Object Model, HypervisorPresent | ConvertTo-Json -Compress`); err != nil {
		logger.Warn("hardware collector: virtualization query failed", "error", err)
	} else {
		hw.Virtualization = parseWinVirtualization(out)
	}

	return hw, nil
}

// runPSCtx executes a PowerShell command with context and returns trimmed stdout.
func runPSCtx(ctx context.Context, cmd string) (string, error) {
	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
```

- [ ] **Step 2: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: Compiles.

- [ ] **Step 3: Commit**

```bash
git add internal/agent/inventory/hardware_windows.go
git commit -m "feat(agent): Windows hardware collector with 10 CIM subsystems"
```

---

## Task 7: Windows Services Collector (§3)

**Files:**
- Create: `internal/agent/inventory/services_windows_parse.go` (no build tag)
- Create: `internal/agent/inventory/services_windows_parse_test.go` (no build tag)
- Modify: `internal/agent/inventory/services_windows.go`

- [ ] **Step 1: Write service parse tests**

Create `internal/agent/inventory/services_windows_parse_test.go`:

```go
package inventory

import (
	"os"
	"testing"
)

func TestParseWinServices(t *testing.T) {
	data, err := os.ReadFile("testdata/windows/services.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	services := parseWinServices(string(data))

	if len(services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(services))
	}

	// wuauserv — Running, Manual start.
	wu := services[0]
	if wu.Name != "wuauserv" {
		t.Errorf("services[0].Name = %q", wu.Name)
	}
	if wu.Description != "Windows Update" {
		t.Errorf("services[0].Description = %q", wu.Description)
	}
	if wu.ActiveState != "active" {
		t.Errorf("services[0].ActiveState = %q, want active", wu.ActiveState)
	}
	if wu.SubState != "running" {
		t.Errorf("services[0].SubState = %q, want running", wu.SubState)
	}
	if wu.Enabled {
		t.Error("services[0].Enabled should be false (Manual start)")
	}

	// WinDefend — Running, Automatic start.
	wd := services[1]
	if !wd.Enabled {
		t.Error("services[1].Enabled should be true (Automatic)")
	}
	if wd.Category != "Security" {
		t.Errorf("services[1].Category = %q, want Security", wd.Category)
	}

	// Spooler — Stopped, Manual start.
	sp := services[2]
	if sp.ActiveState != "inactive" {
		t.Errorf("services[2].ActiveState = %q, want inactive", sp.ActiveState)
	}
	if sp.SubState != "dead" {
		t.Errorf("services[2].SubState = %q, want dead", sp.SubState)
	}

	// MSSQLSERVER — Running, Automatic.
	ms := services[3]
	if ms.Category != "Database" {
		t.Errorf("services[3].Category = %q, want Database", ms.Category)
	}
}

func TestParseWinServices_Empty(t *testing.T) {
	services := parseWinServices("")
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/agent/inventory/ -run TestParseWinServices -v`
Expected: FAIL — `parseWinServices` not defined.

- [ ] **Step 3: Implement service parser**

Create `internal/agent/inventory/services_windows_parse.go`:

```go
package inventory

import (
	"encoding/json"
	"strings"
)

type winServiceEntry struct {
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`
	Status      int    `json:"Status"`
	StartType   int    `json:"StartType"`
}

// parseWinServices parses Get-Service JSON output into ServiceInfo slices.
// Status: 1=Stopped, 2=StartPending, 3=StopPending, 4=Running, 5=ContinuePending, 6=PausePending, 7=Paused.
// StartType: 0=Boot, 1=System, 2=Automatic, 3=Manual, 4=Disabled.
func parseWinServices(data string) []ServiceInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	var entries []winServiceEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return nil
		}
	} else {
		var single winServiceEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return nil
		}
		entries = []winServiceEntry{single}
	}

	services := make([]ServiceInfo, 0, len(entries))
	for _, e := range entries {
		activeState, subState := winServiceState(e.Status)
		svc := ServiceInfo{
			Name:        e.Name,
			Description: e.DisplayName,
			LoadState:   "loaded",
			ActiveState: activeState,
			SubState:    subState,
			Enabled:     e.StartType == 0 || e.StartType == 1 || e.StartType == 2,
			Category:    categorizeWinService(e.Name),
		}
		services = append(services, svc)
	}

	return services
}

func winServiceState(status int) (activeState, subState string) {
	switch status {
	case 4:
		return "active", "running"
	case 1:
		return "inactive", "dead"
	case 7:
		return "inactive", "paused"
	default:
		return "activating", "start-pending"
	}
}

func categorizeWinService(name string) string {
	lower := strings.ToLower(name)
	patterns := []struct {
		matches  []string
		category string
	}{
		{[]string{"wuauserv", "windefend", "securityhealth", "mpssvc", "wscsvc"}, "Security"},
		{[]string{"mssql", "mysql", "postgres", "mongodb", "redis", "valkey"}, "Database"},
		{[]string{"w32time", "dnscache", "winrm", "sshd", "dhcp", "netbt"}, "Network"},
		{[]string{"eventlog", "winmgmt", "diagtrack"}, "Monitoring"},
		{[]string{"docker", "containerd"}, "Container"},
		{[]string{"bits", "wuauserv", "trustedinstaller", "msiserver"}, "Package Management"},
		{[]string{"spooler", "audioendpointbuilder", "audiosrv"}, "Hardware"},
		{[]string{"schedule", "eventlog"}, "Maintenance"},
	}

	for _, group := range patterns {
		for _, p := range group.matches {
			if strings.Contains(lower, p) {
				return group.category
			}
		}
	}
	return "Other"
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/agent/inventory/ -run TestParseWinServices -v`
Expected: All PASS.

- [ ] **Step 5: Implement the services collector**

Rewrite `internal/agent/inventory/services_windows.go`:

```go
//go:build windows

package inventory

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// CollectServices collects Windows service information via Get-Service.
func CollectServices(ctx context.Context, logger *slog.Logger) ([]ServiceInfo, error) {
	const psCmd = `Get-Service | Select-Object Name, DisplayName, Status, StartType | ConvertTo-Json -Compress`

	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
	if err != nil {
		return nil, fmt.Errorf("collect services: Get-Service: %w", err)
	}

	services := parseWinServices(string(bytes.TrimSpace(out)))
	if logger != nil {
		logger.Info("windows services collected", "count", len(services))
	}
	return services, nil
}
```

- [ ] **Step 6: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: Compiles.

- [ ] **Step 7: Commit**

```bash
git add internal/agent/inventory/services_windows_parse.go internal/agent/inventory/services_windows_parse_test.go internal/agent/inventory/services_windows.go
git commit -m "feat(agent): Windows services collector via Get-Service"
```

---

## Task 8: Pre/Post Script Shell Fix (§4)

**Files:**
- Create: `internal/agent/patcher/shell_windows.go`
- Create: `internal/agent/patcher/shell_unix.go`
- Modify: `internal/agent/patcher/patcher.go`

- [ ] **Step 1: Create `shell_unix.go`**

```go
//go:build !windows

package patcher

// scriptShell returns the shell and flag for executing pre/post scripts on Unix.
func scriptShell() (string, string) {
	return "sh", "-c"
}
```

- [ ] **Step 2: Create `shell_windows.go`**

```go
//go:build windows

package patcher

// scriptShell returns the shell and flag for executing pre/post scripts on Windows.
func scriptShell() (string, string) {
	return "powershell.exe", "-NoProfile -NonInteractive -Command"
}
```

- [ ] **Step 3: Update patcher.go to use `scriptShell()`**

In `internal/agent/patcher/patcher.go`, replace the pre-script execution (around line 168):

```go
// Replace:
preResult, err := m.executor.Execute(ctx, "sh", "-c", payload.PreScript)

// With:
shell, shellFlag := scriptShell()
preResult, err := m.executor.Execute(ctx, shell, shellFlag, payload.PreScript)
```

And the post-script execution (around line 300):

```go
// Replace:
postResult, err := m.executor.Execute(ctx, "sh", "-c", payload.PostScript)

// With:
shell, shellFlag := scriptShell()
postResult, err := m.executor.Execute(ctx, shell, shellFlag, payload.PostScript)
```

- [ ] **Step 4: Run existing patcher tests**

Run: `go test ./internal/agent/patcher/ -v -count=1`
Expected: All existing tests pass.

- [ ] **Step 5: Verify cross-compile**

Run: `GOOS=windows GOARCH=amd64 go build ./cmd/agent/...`
Expected: Compiles.

- [ ] **Step 6: Commit**

```bash
git add internal/agent/patcher/shell_unix.go internal/agent/patcher/shell_windows.go internal/agent/patcher/patcher.go
git commit -m "fix(agent): use platform-specific shell for pre/post scripts"
```

---

## Task 9: Final Cross-Compile and Full Test Run

**Files:** None new — verification only.

- [ ] **Step 1: Full Go test suite**

Run: `go test ./cmd/agent/... ./internal/agent/... -v -count=1`
Expected: All pass.

- [ ] **Step 2: Cross-compile all agent targets**

Run: `make build-agents`
Expected: Builds linux/amd64, linux/arm64, windows/amd64 successfully.

- [ ] **Step 3: Frontend lint**

Run: `cd web && npx eslint src/pages/agent-downloads/AgentDownloadsPage.tsx`
Expected: No errors.

- [ ] **Step 4: Commit any remaining fixes**

If any test or compile issue was found and fixed in previous steps, ensure it's committed.
