//go:build linux

package inventory

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// CollectHardware gathers deep hardware inventory from a Linux endpoint.
// Individual subsystem failures are logged as warnings but do not fail the
// overall collection — partial data is returned.
func CollectHardware(ctx context.Context, logger *slog.Logger) (*HardwareInfo, error) {
	hw := &HardwareInfo{}

	if cpu, err := collectCPU(ctx); err != nil {
		logger.Warn("hardware collector: cpu failed", "error", err)
	} else {
		hw.CPU = *cpu
	}

	if mem, err := collectMemory(ctx); err != nil {
		logger.Warn("hardware collector: memory failed", "error", err)
	} else {
		hw.Memory = *mem
	}

	if mb, err := collectMotherboard(ctx); err != nil {
		logger.Debug("hardware collector: motherboard failed", "error", err)
	} else {
		hw.Motherboard = *mb
	}

	if storage, err := collectStorage(ctx); err != nil {
		logger.Warn("hardware collector: storage failed", "error", err)
	} else {
		hw.Storage = storage
	}

	if gpus, err := collectGPU(ctx); err != nil {
		logger.Warn("hardware collector: gpu failed", "error", err)
	} else {
		hw.GPU = gpus
	}

	if nics, err := collectNetwork(ctx); err != nil {
		logger.Warn("hardware collector: network failed", "error", err)
	} else {
		hw.Network = nics
	}

	if usb, err := collectUSB(ctx); err != nil {
		logger.Warn("hardware collector: usb failed", "error", err)
	} else {
		hw.USB = usb
	}

	if bat, err := collectBattery(); err != nil {
		logger.Warn("hardware collector: battery failed", "error", err)
	} else {
		hw.Battery = *bat
	}

	if tpm, err := collectTPM(); err != nil {
		logger.Warn("hardware collector: tpm failed", "error", err)
	} else {
		hw.TPM = *tpm
	}

	if virt, err := collectVirtualization(ctx); err != nil {
		logger.Warn("hardware collector: virtualization failed", "error", err)
	} else {
		hw.Virtualization = *virt
	}

	return hw, nil
}

// collectCPU parses lscpu key:value output for processor details.
func collectCPU(ctx context.Context) (*CPUInfo, error) {
	out, err := exec.CommandContext(ctx, "lscpu").Output()
	if err != nil {
		return nil, fmt.Errorf("collect cpu: run lscpu: %w", err)
	}

	info := &CPUInfo{}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		key, val, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "Model name":
			info.ModelName = val
		case "Vendor ID":
			info.Vendor = val
		case "CPU family":
			info.Family = val
		case "Model":
			info.Model = val
		case "Stepping":
			info.Stepping = val
		case "Architecture":
			info.Architecture = val
		case "Core(s) per socket":
			info.CoresPerSocket, _ = strconv.Atoi(val)
		case "Thread(s) per core":
			info.ThreadsPerCore, _ = strconv.Atoi(val)
		case "Socket(s)":
			info.Sockets, _ = strconv.Atoi(val)
		case "CPU(s)":
			info.TotalLogical, _ = strconv.Atoi(val)
		case "CPU max MHz":
			info.MaxMHz, _ = strconv.ParseFloat(val, 64)
		case "CPU min MHz":
			info.MinMHz, _ = strconv.ParseFloat(val, 64)
		case "BogoMIPS":
			info.BogoMIPS, _ = strconv.ParseFloat(val, 64)
		case "L1d cache":
			info.CacheL1d = val
		case "L1i cache":
			info.CacheL1i = val
		case "L2 cache":
			info.CacheL2 = val
		case "L3 cache":
			info.CacheL3 = val
		case "Flags":
			info.Flags = strings.Fields(val)
		case "Virtualization type", "Hypervisor vendor":
			if info.VirtType == "" {
				info.VirtType = val
			}
		}
	}

	return info, nil
}

// collectMemory reads /proc/meminfo and parses dmidecode output for DIMM details.
func collectMemory(ctx context.Context) (*MemoryInfo, error) {
	info := &MemoryInfo{}

	// /proc/meminfo is always available on Linux.
	meminfo, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("collect memory: read /proc/meminfo: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(meminfo))
	for scanner.Scan() {
		key, val, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)

		switch key {
		case "MemTotal":
			info.TotalBytes = parseMeminfoKB(val) * 1024
		case "MemAvailable":
			info.AvailableBytes = parseMeminfoKB(val) * 1024
		}
	}

	// dmidecode requires root; best-effort for DIMM details.
	if _, lookErr := exec.LookPath("dmidecode"); lookErr != nil {
		return info, nil
	}

	// Physical Memory Array (type 16) — max capacity and slot count.
	if out, cmdErr := exec.CommandContext(ctx, "dmidecode", "-t", "16").Output(); cmdErr == nil {
		parsePhysicalMemoryArray(out, info)
	}

	// Memory Device (type 17) — per-DIMM details.
	if out, cmdErr := exec.CommandContext(ctx, "dmidecode", "-t", "17").Output(); cmdErr == nil {
		info.DIMMs = parseMemoryDevices(out)
	}

	return info, nil
}

// parseMeminfoKB extracts the numeric kB value from a meminfo line value like "16384 kB".
func parseMeminfoKB(val string) uint64 {
	val = strings.TrimSuffix(val, " kB")
	val = strings.TrimSpace(val)
	n, _ := strconv.ParseUint(val, 10, 64)
	return n
}

// parsePhysicalMemoryArray extracts max capacity and slot count from dmidecode -t 16 output.
func parsePhysicalMemoryArray(data []byte, info *MemoryInfo) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}

		switch key {
		case "Maximum Capacity":
			info.MaxCapacity = val
		case "Number Of Devices":
			info.NumSlots, _ = strconv.Atoi(val)
		case "Error Correction Type":
			info.ErrorCorrection = val
		}
	}
}

// parseMemoryDevices parses dmidecode -t 17 output into DIMM info slices.
// Each "Memory Device" handle block becomes one DIMMInfo entry.
func parseMemoryDevices(data []byte) []DIMMInfo {
	var dimms []DIMMInfo
	var current *DIMMInfo

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// New handle block starts a new DIMM.
		if strings.HasPrefix(line, "Handle ") {
			if current != nil {
				dimms = append(dimms, *current)
			}
			current = nil
			continue
		}

		if strings.Contains(line, "Memory Device") && !strings.Contains(line, "Mapped") {
			current = &DIMMInfo{}
			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)
		key, val, ok := strings.Cut(trimmed, ": ")
		if !ok {
			continue
		}

		switch key {
		case "Locator":
			current.Locator = val
		case "Bank Locator":
			current.BankLocator = val
		case "Size":
			current.SizeMB = parseDIMMSizeMB(val)
		case "Type":
			current.Type = val
		case "Speed":
			current.SpeedMHz = parseFirstInt(val)
		case "Manufacturer":
			current.Manufacturer = val
		case "Serial Number":
			current.SerialNumber = val
		case "Part Number":
			current.PartNumber = strings.TrimSpace(val)
		case "Form Factor":
			current.FormFactor = val
		case "Rank":
			current.Rank = val
		}
	}

	if current != nil {
		dimms = append(dimms, *current)
	}

	return dimms
}

// parseDIMMSizeMB parses a DIMM size string like "8192 MB" or "16 GB" into megabytes.
func parseDIMMSizeMB(val string) int {
	val = strings.TrimSpace(val)
	if strings.EqualFold(val, "No Module Installed") || strings.EqualFold(val, "Not Installed") {
		return 0
	}

	parts := strings.Fields(val)
	if len(parts) < 1 {
		return 0
	}

	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}

	if len(parts) >= 2 {
		switch strings.ToUpper(parts[1]) {
		case "GB":
			return n * 1024
		case "TB":
			return n * 1024 * 1024
		}
	}

	return n // assume MB
}

// parseFirstInt extracts the first integer from a string like "3200 MT/s".
func parseFirstInt(val string) int {
	fields := strings.Fields(val)
	if len(fields) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(fields[0])
	return n
}

// collectMotherboard runs dmidecode to get baseboard and BIOS info.
func collectMotherboard(ctx context.Context) (*MotherboardInfo, error) {
	if _, err := exec.LookPath("dmidecode"); err != nil {
		return nil, fmt.Errorf("collect motherboard: dmidecode not found: %w", err)
	}

	info := &MotherboardInfo{}

	if out, err := exec.CommandContext(ctx, "dmidecode", "-t", "baseboard").Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			key, val, ok := strings.Cut(line, ": ")
			if !ok {
				continue
			}
			switch key {
			case "Manufacturer":
				info.BoardManufacturer = val
			case "Product Name":
				info.BoardProduct = val
			case "Version":
				info.BoardVersion = val
			case "Serial Number":
				info.BoardSerial = val
			}
		}
	}

	if out, err := exec.CommandContext(ctx, "dmidecode", "-t", "bios").Output(); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			key, val, ok := strings.Cut(line, ": ")
			if !ok {
				continue
			}
			switch key {
			case "Vendor":
				info.BIOSVendor = val
			case "Version":
				info.BIOSVersion = val
			case "Release Date":
				info.BIOSReleaseDate = val
			}
		}
	}

	return info, nil
}

// lsblkOutput is the JSON structure returned by lsblk -J.
type lsblkOutput struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       json.Number   `json:"size"`
	Type       string        `json:"type"`
	Model      *string       `json:"model"`
	Serial     *string       `json:"serial"`
	FSType     *string       `json:"fstype"`
	MountPoint *string       `json:"mountpoint"`
	Tran       *string       `json:"tran"`
	Rota       *bool         `json:"rota"`
	Children   []lsblkDevice `json:"children"`
}

// collectStorage uses lsblk and smartctl to enumerate storage devices.
func collectStorage(ctx context.Context) ([]StorageDevice, error) {
	out, err := exec.CommandContext(ctx, "lsblk", "-J", "-b", "-o",
		"NAME,SIZE,TYPE,MODEL,SERIAL,FSTYPE,MOUNTPOINT,TRAN,ROTA").Output()
	if err != nil {
		return nil, fmt.Errorf("collect storage: run lsblk: %w", err)
	}

	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("collect storage: parse lsblk json: %w", err)
	}

	var devices []StorageDevice
	for _, blk := range parsed.BlockDevices {
		if blk.Type != "disk" {
			continue
		}

		dev := StorageDevice{
			Name:      blk.Name,
			Model:     derefStr(blk.Model),
			Serial:    derefStr(blk.Serial),
			Transport: derefStr(blk.Tran),
		}

		if sz, err := blk.Size.Int64(); err == nil {
			dev.SizeBytes = uint64(sz)
		}

		// Determine disk type.
		dev.Type = classifyDiskType(blk.Name, blk.Rota, dev.Transport)

		// Collect partitions.
		for _, child := range blk.Children {
			if child.Type != "part" {
				continue
			}
			part := PartitionInfo{
				Name:       child.Name,
				FSType:     derefStr(child.FSType),
				MountPoint: derefStr(child.MountPoint),
			}
			if sz, err := child.Size.Int64(); err == nil {
				part.SizeBytes = uint64(sz)
			}
			if part.MountPoint != "" {
				part.UsagePct = mountUsagePct(ctx, part.MountPoint)
			}
			dev.Partitions = append(dev.Partitions, part)
		}

		// SMART data is best-effort. Use -i -H -A to get info, health, AND attributes (temperature).
		// smartctl uses bitmask exit codes — non-zero doesn't mean no data,
		// it means the disk has issues. Parse output when there's an *exec.ExitError.
		smartOut, smartErr := exec.CommandContext(ctx, "smartctl", "-i", "-H", "-A",
			"/dev/"+blk.Name).CombinedOutput()
		if smartErr == nil {
			parseSmartctl(smartOut, &dev)
		} else {
			var exitErr *exec.ExitError
			if errors.As(smartErr, &exitErr) && len(smartOut) > 0 {
				parseSmartctl(smartOut, &dev) // bitmask exit; output still useful
			}
			if dev.SmartStatus == "" {
				dev.SmartStatus = "N/A"
			}
		}

		devices = append(devices, dev)
	}

	return devices, nil
}

// classifyDiskType determines if a disk is nvme, ssd, or hdd.
func classifyDiskType(name string, rota *bool, transport string) string {
	if strings.HasPrefix(name, "nvme") || transport == "nvme" {
		return "nvme"
	}

	// Check /sys rotational flag.
	rotaPath := "/sys/block/" + name + "/queue/rotational"
	if data, err := os.ReadFile(rotaPath); err == nil {
		val := strings.TrimSpace(string(data))
		if val == "0" {
			return "ssd"
		}
		if val == "1" {
			return "hdd"
		}
	}

	// Fallback to lsblk rota field.
	if rota != nil {
		if !*rota {
			return "ssd"
		}
		return "hdd"
	}

	return "unknown"
}

// parseSmartctl extracts firmware version, SMART health, and temperature from smartctl output.
func parseSmartctl(data []byte, dev *StorageDevice) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		if key, val, ok := strings.Cut(line, ":"); ok {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)

			switch key {
			case "Firmware Version":
				dev.FirmwareVersion = val
			case "SMART overall-health self-assessment test result":
				dev.SmartStatus = val
			case "SMART Health Status": // NVMe variant
				dev.SmartStatus = val
			}
		}

		// Temperature line varies: "Temperature:   35 Celsius" or attribute table row.
		if strings.Contains(line, "Temperature") && strings.Contains(line, "Celsius") {
			if t := extractTemperature(line); t > 0 {
				dev.TempCelsius = t
			}
		}
	}
}

// extractTemperature pulls the numeric Celsius value from a smartctl temperature line.
func extractTemperature(line string) int {
	re := regexp.MustCompile(`(\d+)\s*Celsius`)
	m := re.FindStringSubmatch(line)
	if len(m) >= 2 {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// mountUsagePct calculates disk usage percentage for a mount point using syscall-free approach.
func mountUsagePct(ctx context.Context, mountpoint string) int {
	// Use df to get usage; avoids importing syscall.
	out, err := exec.CommandContext(ctx, "df", "--output=pcent", mountpoint).Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0
	}
	pct := strings.TrimSpace(lines[len(lines)-1])
	pct = strings.TrimSuffix(pct, "%")
	n, _ := strconv.Atoi(pct)
	return n
}

// collectGPU tries nvidia-smi first (with utilization), then falls back to lspci.
func collectGPU(ctx context.Context) ([]GPUInfo, error) {
	// Try nvidia-smi for NVIDIA GPUs with detailed info + utilization.
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		out, err := exec.CommandContext(ctx, "nvidia-smi",
			"--query-gpu=name,memory.total,driver_version,gpu_bus_id,utilization.gpu",
			"--format=csv,noheader,nounits").Output()
		if err == nil {
			gpus := parseNvidiaSMI(out)
			if len(gpus) > 0 {
				return gpus, nil
			}
		}
	}

	// Fallback: lspci for any VGA/3D controllers.
	out, err := exec.CommandContext(ctx, "lspci").Output()
	if err != nil {
		return nil, fmt.Errorf("collect gpu: run lspci: %w", err)
	}

	gpus := parseLSPCIForGPU(out)
	populateAMDGPUUsage(gpus)
	return gpus, nil
}

// parseNvidiaSMI parses nvidia-smi CSV output including utilization.gpu.
func parseNvidiaSMI(data []byte) []GPUInfo {
	var gpus []GPUInfo
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ", ")
		if len(fields) < 4 {
			continue
		}

		vram, _ := strconv.Atoi(strings.TrimSpace(fields[1]))
		gpu := GPUInfo{
			Model:         strings.TrimSpace(fields[0]),
			VRAMMB:        vram,
			DriverVersion: strings.TrimSpace(fields[2]),
			PCISlot:       strings.TrimSpace(fields[3]),
		}
		if len(fields) >= 5 {
			gpu.UsagePct, _ = strconv.Atoi(strings.TrimSpace(fields[4]))
		}
		gpus = append(gpus, gpu)
	}
	return gpus
}

// populateAMDGPUUsage reads /sys/class/drm/card*/device/gpu_busy_percent for AMD GPUs
// and matches them to collected GPUInfo entries by PCI slot.
func populateAMDGPUUsage(gpus []GPUInfo) {
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 4 || name[:4] != "card" || strings.ContainsRune(name[4:], '-') {
			continue
		}
		data, err := os.ReadFile("/sys/class/drm/" + name + "/device/gpu_busy_percent")
		if err != nil {
			continue
		}
		pct, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}
		// Resolve PCI slot from sysfs symlink: .../0000:01:00.0 → "01:00.0"
		linkTarget, err := os.Readlink("/sys/class/drm/" + name + "/device")
		if err != nil {
			continue
		}
		parts := strings.Split(filepath.Base(linkTarget), ":")
		if len(parts) < 3 {
			continue
		}
		slot := parts[1] + ":" + parts[2]
		for i := range gpus {
			if strings.EqualFold(gpus[i].PCISlot, slot) {
				gpus[i].UsagePct = pct
				break
			}
		}
	}
}

// parseLSPCIForGPU extracts VGA and 3D controller entries from lspci output.
func parseLSPCIForGPU(data []byte) []GPUInfo {
	var gpus []GPUInfo
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// lspci format: "XX:XX.X Class: Description"
		if !strings.Contains(line, "VGA") && !strings.Contains(line, "3D controller") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		slot := parts[0]
		// Remove class prefix like "VGA compatible controller: "
		desc := parts[1]
		if idx := strings.Index(desc, ": "); idx >= 0 {
			desc = desc[idx+2:]
		}

		gpus = append(gpus, GPUInfo{
			Model:   strings.TrimSpace(desc),
			PCISlot: slot,
		})
	}
	return gpus
}

// ipAddrEntry represents a single interface from `ip -j addr show` JSON output.
type ipAddrEntry struct {
	IFName    string         `json:"ifname"`
	Address   string         `json:"address"`
	MTU       int            `json:"mtu"`
	OperState string         `json:"operstate"`
	AddrInfo  []ipAddrDetail `json:"addr_info"`
}

type ipAddrDetail struct {
	Family    string `json:"family"`
	Local     string `json:"local"`
	PrefixLen int    `json:"prefixlen"`
}

// collectNetwork uses `ip -j addr show` to enumerate network interfaces.
func collectNetwork(ctx context.Context) ([]NetworkInfo, error) {
	out, err := exec.CommandContext(ctx, "ip", "-j", "addr", "show").Output()
	if err != nil {
		return nil, fmt.Errorf("collect network: run ip addr: %w", err)
	}

	var entries []ipAddrEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("collect network: parse ip json: %w", err)
	}

	var nics []NetworkInfo
	for _, e := range entries {
		if e.IFName == "lo" {
			continue
		}

		nic := NetworkInfo{
			Name:       e.IFName,
			MACAddress: e.Address,
			MTU:        e.MTU,
			State:      strings.ToLower(e.OperState),
			Type:       classifyNetType(e.IFName),
		}

		for _, addr := range e.AddrInfo {
			ip := IPAddress{
				Address:   addr.Local,
				PrefixLen: addr.PrefixLen,
			}
			switch addr.Family {
			case "inet":
				nic.IPv4Addresses = append(nic.IPv4Addresses, ip)
			case "inet6":
				nic.IPv6Addresses = append(nic.IPv6Addresses, ip)
			}
		}

		// Speed from sysfs (returns -1 if not applicable, e.g. for virtual interfaces).
		speedPath := "/sys/class/net/" + e.IFName + "/speed"
		if data, readErr := os.ReadFile(speedPath); readErr == nil {
			if spd, parseErr := strconv.Atoi(strings.TrimSpace(string(data))); parseErr == nil && spd > 0 {
				nic.SpeedMbps = spd
			}
		}

		// Driver from sysfs symlink.
		driverPath := "/sys/class/net/" + e.IFName + "/device/driver"
		if target, linkErr := os.Readlink(driverPath); linkErr == nil {
			nic.Driver = filepath.Base(target)
		}

		nics = append(nics, nic)
	}

	return nics, nil
}

// classifyNetType infers network interface type from its name prefix.
func classifyNetType(name string) string {
	switch {
	case strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "virbr"):
		return "virtual"
	case strings.HasPrefix(name, "br"):
		return "bridge"
	case strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "wlan"):
		return "wifi"
	case strings.HasPrefix(name, "en") || strings.HasPrefix(name, "eth"):
		return "ethernet"
	case strings.HasPrefix(name, "bond"):
		return "bond"
	case strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap"):
		return "virtual"
	default:
		return "other"
	}
}

// lsusbRe matches lsusb output lines: "Bus XXX Device YYY: ID VVVV:PPPP Description".
var lsusbRe = regexp.MustCompile(`^Bus\s+(\d+)\s+Device\s+(\d+):\s+ID\s+([0-9a-fA-F]{4}):([0-9a-fA-F]{4})\s+(.*)$`)

// collectUSB parses lsusb output to enumerate USB devices.
func collectUSB(ctx context.Context) ([]USBDevice, error) {
	out, err := exec.CommandContext(ctx, "lsusb").Output()
	if err != nil {
		return nil, fmt.Errorf("collect usb: run lsusb: %w", err)
	}

	var devices []USBDevice
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		m := lsusbRe.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		devices = append(devices, USBDevice{
			Bus:         m[1],
			DeviceNum:   m[2],
			VendorID:    m[3],
			ProductID:   m[4],
			Description: strings.TrimSpace(m[5]),
		})
	}

	return devices, nil
}

// collectBattery reads battery info from /sys/class/power_supply/BAT*.
func collectBattery() (*BatteryInfo, error) {
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*")
	if err != nil {
		return nil, fmt.Errorf("collect battery: glob: %w", err)
	}

	if len(matches) == 0 {
		return &BatteryInfo{Present: false}, nil
	}

	bat := matches[0] // Use first battery.
	info := &BatteryInfo{Present: true}

	info.Status = readSysfsString(filepath.Join(bat, "status"))
	info.Technology = readSysfsString(filepath.Join(bat, "technology"))
	info.CapacityPct = readSysfsInt(filepath.Join(bat, "capacity"))
	info.CycleCount = readSysfsInt(filepath.Join(bat, "cycle_count"))

	// Energy values are in microwatt-hours; convert to watt-hours.
	if eNow := readSysfsInt(filepath.Join(bat, "energy_full")); eNow > 0 {
		info.EnergyFullWh = float64(eNow) / 1e6
	}
	if eDesign := readSysfsInt(filepath.Join(bat, "energy_full_design")); eDesign > 0 {
		info.EnergyDesignWh = float64(eDesign) / 1e6
	}

	// Calculate health percentage.
	if info.EnergyDesignWh > 0 && info.EnergyFullWh > 0 {
		info.HealthPct = int(math.Round(info.EnergyFullWh / info.EnergyDesignWh * 100))
	}

	return info, nil
}

// collectTPM checks for TPM presence via sysfs.
func collectTPM() (*TPMInfo, error) {
	info := &TPMInfo{}

	if _, err := os.Stat("/sys/class/tpm/tpm0"); err != nil {
		return info, nil
	}

	info.Present = true

	// Try to read TPM version from the device description or caps.
	versionPath := "/sys/class/tpm/tpm0/tpm_version_major"
	if v := readSysfsString(versionPath); v != "" {
		info.Version = v + ".0"
	} else {
		// Fallback: check /sys/class/tpm/tpm0/device/description or caps.
		capsPath := "/sys/class/tpm/tpm0/caps"
		if data, err := os.ReadFile(capsPath); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(data))
			for scanner.Scan() {
				if key, val, ok := strings.Cut(scanner.Text(), ": "); ok {
					if strings.Contains(key, "TCG version") {
						info.Version = val
						break
					}
				}
			}
		}
	}

	return info, nil
}

// collectVirtualization uses systemd-detect-virt to determine if the system is virtualized.
func collectVirtualization(ctx context.Context) (*VirtInfo, error) {
	info := &VirtInfo{}

	out, err := exec.CommandContext(ctx, "systemd-detect-virt").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return info, nil // exit 1 = not virtualized
		}
		// Genuine error — still return non-virtual but log it.
		slog.Warn("collect virtualization: systemd-detect-virt failed", "error", err)
		return info, nil
	}

	result := strings.TrimSpace(string(out))
	if result == "" || result == "none" {
		return info, nil
	}

	info.IsVirtual = true
	info.HypervisorType = result

	return info, nil
}

// readSysfsString reads a sysfs file and returns its trimmed content.
func readSysfsString(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// readSysfsInt reads a sysfs file and parses its content as an integer.
func readSysfsInt(path string) int {
	s := readSysfsString(path)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

// derefStr safely dereferences a *string, returning "" for nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}
