//go:build darwin

package inventory

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/bits"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// CollectHardware gathers deep hardware inventory from a macOS endpoint.
// Individual subsystem failures are logged as warnings but do not fail the
// overall collection — partial data is returned.
func CollectHardware(ctx context.Context, logger *slog.Logger) (*HardwareInfo, error) {
	hw := &HardwareInfo{}

	if cpu, err := collectDarwinCPU(ctx); err != nil {
		logger.Warn("hardware collector: cpu failed", "error", err)
	} else {
		hw.CPU = *cpu
	}

	if mem, err := collectDarwinMemory(ctx); err != nil {
		logger.Warn("hardware collector: memory failed", "error", err)
	} else {
		hw.Memory = *mem
	}

	if mb, err := collectDarwinMotherboard(ctx); err != nil {
		logger.Debug("hardware collector: motherboard failed", "error", err)
	} else {
		hw.Motherboard = *mb
	}

	if storage, err := collectDarwinStorage(ctx); err != nil {
		logger.Warn("hardware collector: storage failed", "error", err)
	} else {
		hw.Storage = storage
	}

	if gpus, err := collectDarwinGPU(ctx); err != nil {
		logger.Warn("hardware collector: gpu failed", "error", err)
	} else {
		hw.GPU = gpus
	}

	if nics, err := collectDarwinNetwork(ctx); err != nil {
		logger.Warn("hardware collector: network failed", "error", err)
	} else {
		hw.Network = nics
	}

	if usb, err := collectDarwinUSB(ctx); err != nil {
		logger.Warn("hardware collector: usb failed", "error", err)
	} else {
		hw.USB = usb
	}

	if bat, err := collectDarwinBattery(ctx); err != nil {
		logger.Warn("hardware collector: battery failed", "error", err)
	} else {
		hw.Battery = *bat
	}

	// TPM is not present on macOS (Apple uses Secure Enclave instead).
	// Leave hw.TPM as zero value (Present: false).

	if virt, err := collectDarwinVirtualization(ctx); err != nil {
		logger.Warn("hardware collector: virtualization failed", "error", err)
	} else {
		hw.Virtualization = *virt
	}

	return hw, nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// sysctlString runs `sysctl -n <key>` and returns the trimmed output.
func sysctlString(ctx context.Context, key string) (string, error) {
	out, err := exec.CommandContext(ctx, "sysctl", "-n", key).Output()
	if err != nil {
		return "", fmt.Errorf("sysctl %s: %w", key, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// sysctlUint64 runs sysctl and parses the result as uint64.
func sysctlUint64(ctx context.Context, key string) (uint64, error) {
	s, err := sysctlString(ctx, key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("sysctl %s: parse uint64 %q: %w", key, s, err)
	}
	return n, nil
}

// sysctlInt runs sysctl and parses the result as int.
func sysctlInt(ctx context.Context, key string) (int, error) {
	s, err := sysctlString(ctx, key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("sysctl %s: parse int %q: %w", key, s, err)
	}
	return n, nil
}

// runSystemProfiler runs `system_profiler <dataType> -json` and unmarshals
// the JSON output into result.
func runSystemProfiler(ctx context.Context, dataType string, result interface{}) error {
	out, err := exec.CommandContext(ctx, "system_profiler", dataType, "-json").Output()
	if err != nil {
		return fmt.Errorf("system_profiler %s: %w", dataType, err)
	}
	if err := json.Unmarshal(out, result); err != nil {
		return fmt.Errorf("system_profiler %s: parse json: %w", dataType, err)
	}
	return nil
}

// formatCacheSize converts a byte count to a human-readable cache size string.
// Examples: 65536 -> "64 KiB", 4194304 -> "4 MiB", 16777216 -> "16 MiB".
func formatCacheSize(b uint64) string {
	if b == 0 {
		return ""
	}
	const (
		kib = 1024
		mib = 1024 * 1024
		gib = 1024 * 1024 * 1024
	)
	switch {
	case b >= gib && b%gib == 0:
		return fmt.Sprintf("%d GiB", b/gib)
	case b >= mib && b%mib == 0:
		return fmt.Sprintf("%d MiB", b/mib)
	case b >= kib && b%kib == 0:
		return fmt.Sprintf("%d KiB", b/kib)
	case b >= gib:
		return fmt.Sprintf("%.1f GiB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.1f MiB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.1f KiB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// mountUsagePctDarwin uses `df -P` to get usage percentage for a mount point.
// macOS does not support `df --output=pcent`, so we parse POSIX output.
func mountUsagePctDarwin(ctx context.Context, mountpoint string) int {
	out, err := exec.CommandContext(ctx, "df", "-P", mountpoint).Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0
	}
	// POSIX df -P: Filesystem 1024-blocks Used Available Capacity Mounted-on
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 5 {
		return 0
	}
	pct := strings.TrimSuffix(fields[4], "%")
	n, _ := strconv.Atoi(pct)
	return n
}

// parseHexNetmask converts a hex netmask like "0xffffff00" to a prefix length (24).
func parseHexNetmask(hex string) int {
	hex = strings.TrimPrefix(hex, "0x")
	hex = strings.TrimPrefix(hex, "0X")
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0
	}
	return bits.OnesCount32(uint32(n))
}

// ---------------------------------------------------------------------------
// CPU
// ---------------------------------------------------------------------------

func collectDarwinCPU(ctx context.Context) (*CPUInfo, error) {
	info := &CPUInfo{}

	brand, err := sysctlString(ctx, "machdep.cpu.brand_string")
	if err != nil {
		return nil, fmt.Errorf("collect cpu: sysctl brand_string: %w", err)
	}
	info.ModelName = brand

	if vendor, err := sysctlString(ctx, "machdep.cpu.vendor"); err == nil {
		info.Vendor = vendor
	} else {
		// Apple Silicon does not expose machdep.cpu.vendor.
		info.Vendor = "Apple"
	}

	info.Architecture = runtime.GOARCH

	if cores, err := sysctlInt(ctx, "hw.physicalcpu"); err == nil {
		info.CoresPerSocket = cores
	}

	if logical, err := sysctlInt(ctx, "hw.logicalcpu"); err == nil {
		info.TotalLogical = logical
	}

	info.Sockets = 1

	if info.CoresPerSocket > 0 && info.TotalLogical > 0 {
		info.ThreadsPerCore = info.TotalLogical / info.CoresPerSocket
	}
	if info.ThreadsPerCore == 0 {
		info.ThreadsPerCore = 1
	}

	// Cache sizes.
	if v, err := sysctlUint64(ctx, "hw.l1dcachesize"); err == nil {
		info.CacheL1d = formatCacheSize(v)
	}
	if v, err := sysctlUint64(ctx, "hw.l1icachesize"); err == nil {
		info.CacheL1i = formatCacheSize(v)
	}
	if v, err := sysctlUint64(ctx, "hw.l2cachesize"); err == nil {
		info.CacheL2 = formatCacheSize(v)
	}
	if v, err := sysctlUint64(ctx, "hw.l3cachesize"); err == nil {
		info.CacheL3 = formatCacheSize(v)
	}

	// Max frequency (Hz → MHz). hw.cpufrequency_max doesn't exist on Apple Silicon.
	// Fall back to per-performance-level sysctl keys (perflevel0 = P-cores, perflevel1 = E-cores).
	if hz, err := sysctlUint64(ctx, "hw.cpufrequency_max"); err == nil && hz > 0 {
		info.MaxMHz = float64(hz) / 1e6
	} else if mhz, err := sysctlUint64(ctx, "hw.perflevel0.cpufreq_max"); err == nil && mhz > 0 {
		// Apple Silicon: perflevel0 is P-cores, value is already in MHz.
		info.MaxMHz = float64(mhz)
	}

	// Min frequency: on Apple Silicon, E-core max freq serves as the system minimum.
	if info.MinMHz == 0 {
		if mhz, err := sysctlUint64(ctx, "hw.perflevel1.cpufreq_max"); err == nil && mhz > 0 {
			info.MinMHz = float64(mhz)
		}
	}

	// Apple Silicon fallback: powermetrics (root) > lookup table > leave as 0.
	fillAppleSiliconFreq(ctx, info)

	// CPU flags (Intel only; fails gracefully on Apple Silicon).
	if flags, err := sysctlString(ctx, "machdep.cpu.features"); err == nil {
		info.Flags = parseCPUFlags(flags)
	}

	return info, nil
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

func collectDarwinMemory(ctx context.Context) (*MemoryInfo, error) {
	info := &MemoryInfo{}

	total, err := sysctlUint64(ctx, "hw.memsize")
	if err != nil {
		return nil, fmt.Errorf("collect memory: sysctl hw.memsize: %w", err)
	}
	info.TotalBytes = total

	// Format MaxCapacity in GB.
	gb := total / (1024 * 1024 * 1024)
	if gb > 0 {
		info.MaxCapacity = fmt.Sprintf("%d GB", gb)
	}

	// Parse vm_stat for available memory.
	out, err := exec.CommandContext(ctx, "vm_stat").Output()
	if err != nil {
		return info, nil // best-effort
	}

	pageSize, pages := parseVmStat(out)
	if pageSize > 0 {
		active := pages["pages active"] * pageSize
		wired := pages["pages wired down"] * pageSize
		compressor := pages["pages occupied by compressor"] * pageSize
		used := active + wired + compressor
		if info.TotalBytes > used {
			info.AvailableBytes = info.TotalBytes - used
		} else {
			info.AvailableBytes = pages["pages free"] * pageSize
		}
	}

	// Collect DIMM/memory module info from system_profiler.
	var memData spMemoryData
	if spErr := runSystemProfiler(ctx, "SPMemoryDataType", &memData); spErr == nil && len(memData.SPMemoryDataType) > 0 {
		for _, entry := range memData.SPMemoryDataType {
			// Apple Silicon unified memory shows as a single entry.
			// Intel Macs show per-DIMM entries with slot names.
			dimm := DIMMInfo{
				Manufacturer: entry.Manufacturer,
				Type:         entry.Type,
			}

			// Parse size from entry (e.g., "16 GB" or "8 GB").
			if entry.Size != "" {
				dimm.SizeMB = parseMemorySizeMB(entry.Size)
			}

			// Per-slot info (Intel Macs).
			if entry.SlotName != "" {
				dimm.Locator = entry.SlotName
			}

			// Speed (e.g., "6400 MHz" or "2400 MHz").
			if entry.Speed != "" {
				dimm.SpeedMHz = parseFirstInt(entry.Speed)
			}

			// Apple Silicon fallback: speed may be in spdram_speed (e.g., "LPDDR5-6400").
			if dimm.SpeedMHz == 0 && entry.SPDRAMSpeed != "" {
				dimm.SpeedMHz = parseSPDRAMSpeed(entry.SPDRAMSpeed)
			}

			// JEDEC standard fallback: if speed is still unknown, look up by memory type.
			if dimm.SpeedMHz == 0 && dimm.Type != "" {
				if jedecSpeed, ok := jedecMemorySpeeds[strings.ToUpper(strings.TrimSpace(dimm.Type))]; ok {
					dimm.SpeedMHz = jedecSpeed
				}
			}

			// Serial and part number.
			dimm.SerialNumber = entry.SerialNumber
			dimm.PartNumber = entry.PartNumber

			info.DIMMs = append(info.DIMMs, dimm)
		}
		info.NumSlots = len(info.DIMMs)
	}

	return info, nil
}

// spMemoryData is the JSON structure from `system_profiler SPMemoryDataType -json`.
type spMemoryData struct {
	SPMemoryDataType []spMemoryEntry `json:"SPMemoryDataType"`
}

type spMemoryEntry struct {
	Size         string `json:"dimm_size"` // e.g., "16 GB" (Apple Silicon uses dimm_size)
	Type         string `json:"dimm_type"` // e.g., "LPDDR5", "DDR4"
	Manufacturer string `json:"dimm_manufacturer"`
	SlotName     string `json:"_name"`      // e.g., "DIMM0/ChannelA" or "Memory"
	Speed        string `json:"dimm_speed"` // e.g., "6400 MHz" (Intel Macs)
	SerialNumber string `json:"dimm_serial_number"`
	PartNumber   string `json:"dimm_part_number"`

	// Apple Silicon may report speed under "spdram_speed" instead of "dimm_speed".
	SPDRAMSpeed string `json:"spdram_speed"` // e.g., "LPDDR5-6400"
}

// parseMemorySizeMB parses a memory size string like "16 GB" or "8192 MB" into megabytes.
func parseMemorySizeMB(s string) int {
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return 0
	}
	n := parseFirstInt(parts[0])
	switch strings.ToUpper(parts[1]) {
	case "TB":
		return n * 1024 * 1024
	case "GB":
		return n * 1024
	default:
		return n // assume MB
	}
}

// parseFirstInt extracts the first integer from a string like "6400 MHz".
func parseFirstInt(s string) int {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(fields[0])
	return n
}

// jedecMemorySpeeds maps memory type strings to their JEDEC standard speeds in
// MHz. Used as a fallback when system_profiler does not report a numeric speed
// (common on Apple Silicon).
var jedecMemorySpeeds = map[string]int{
	"LPDDR5":  6400,
	"LPDDR5X": 7500,
	"LPDDR4X": 4266,
	"LPDDR4":  3200,
	"DDR5":    4800,
	"DDR4":    3200,
	"DDR3":    1600,
}

// spdramSpeedRe extracts the trailing numeric speed from strings like "LPDDR5-6400".
var spdramSpeedRe = regexp.MustCompile(`-(\d+)\s*$`)

// parseSPDRAMSpeed extracts speed in MHz from an spdram_speed string like "LPDDR5-6400".
func parseSPDRAMSpeed(s string) int {
	// Try to extract the number after the last hyphen (e.g., "LPDDR5-6400" → 6400).
	m := spdramSpeedRe.FindStringSubmatch(s)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// ---------------------------------------------------------------------------
// Motherboard
// ---------------------------------------------------------------------------

type spHardwareData struct {
	SPHardwareDataType []spHardwareEntry `json:"SPHardwareDataType"`
}

type spHardwareEntry struct {
	MachineName           string `json:"machine_name"`
	MachineModel          string `json:"machine_model"`
	ChipType              string `json:"chip_type"`
	SerialNumber          string `json:"serial_number_system"`
	BootROMVersion        string `json:"boot_rom_version"`
	PhysicalMemory        string `json:"physical_memory"`
	ModelNumber           string `json:"model_number"`
	CurrentProcessorSpeed string `json:"current_processor_speed"`
}

func collectDarwinMotherboard(ctx context.Context) (*MotherboardInfo, error) {
	var data spHardwareData
	if err := runSystemProfiler(ctx, "SPHardwareDataType", &data); err != nil {
		return nil, fmt.Errorf("collect motherboard: %w", err)
	}

	info := &MotherboardInfo{
		BoardManufacturer: "Apple",
		BIOSVendor:        "Apple",
	}

	if len(data.SPHardwareDataType) > 0 {
		entry := data.SPHardwareDataType[0]
		info.BoardProduct = entry.MachineModel
		info.BoardSerial = entry.SerialNumber
		info.BIOSVersion = entry.BootROMVersion
	}

	return info, nil
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

type spStorageData struct {
	SPStorageDataType []spStorageVolume `json:"SPStorageDataType"`
}

type spStorageVolume struct {
	Name           string          `json:"_name"`
	BSDName        string          `json:"bsd_name"`
	FileSystem     string          `json:"file_system"`
	FreeSpaceBytes json.Number     `json:"free_space_in_bytes"`
	MountPoint     string          `json:"mount_point"`
	SizeBytes      json.Number     `json:"size_in_bytes"`
	PhysicalDrive  spPhysicalDrive `json:"physical_drive"`
}

type spPhysicalDrive struct {
	DeviceName   string `json:"device_name"`
	IsInternal   string `json:"is_internal_disk"`
	MediumType   string `json:"medium_type"`
	PartitionMap string `json:"partition_map_type"`
	Protocol     string `json:"protocol"`
}

func collectDarwinStorage(ctx context.Context) ([]StorageDevice, error) {
	var data spStorageData
	if err := runSystemProfiler(ctx, "SPStorageDataType", &data); err != nil {
		return nil, fmt.Errorf("collect storage: %w", err)
	}

	devices := parseSPStorageDataType(ctx, data)

	// Enrich with diskutil info for model and serial.
	for i := range devices {
		for _, part := range devices[i].Partitions {
			if part.Name == "" {
				continue
			}
			out, err := exec.CommandContext(ctx, "diskutil", "info", "/dev/"+part.Name).Output()
			if err != nil {
				continue
			}
			model, serial, smart := parseDiskutilInfo(out)
			if model != "" && devices[i].Model == devices[i].Name {
				devices[i].Model = model
			}
			if serial != "" && devices[i].Serial == "" {
				devices[i].Serial = serial
			}
			if smart != "" && devices[i].SmartStatus == "" {
				devices[i].SmartStatus = smart
			}
			break // Only need first partition per device.
		}
	}

	enrichStorageFromNVMe(ctx, devices)

	return devices, nil
}

// ---------------------------------------------------------------------------
// NVMe enrichment (privileged)
// ---------------------------------------------------------------------------

// spNVMeData mirrors the top-level JSON from `system_profiler SPNVMeDataType -json`.
type spNVMeData struct {
	SPNVMeDataType []struct {
		Items []spNVMeItem `json:"_items"`
	} `json:"SPNVMeDataType"`
}

// spNVMeItem represents a single NVMe device from system_profiler.
type spNVMeItem struct {
	Name           string `json:"_name"`
	BSDName        string `json:"bsd_name"`
	DeviceModel    string `json:"device_model"`
	DeviceSerial   string `json:"device_serial"`
	DeviceRevision string `json:"device_revision"`
	SizeInBytes    int64  `json:"size_in_bytes"`
	SmartStatus    string `json:"smart_status"`
}

// enrichStorageFromNVMe uses `system_profiler SPNVMeDataType -json` to fill in
// Serial and FirmwareVersion fields on NVMe storage devices. This command
// requires root privileges to expose serial numbers, so it is skipped when
// running as a non-root user. All failures are handled gracefully — this
// function never returns an error.
func enrichStorageFromNVMe(ctx context.Context, devices []StorageDevice) {
	if os.Geteuid() != 0 {
		return
	}

	var data spNVMeData
	if err := runSystemProfiler(ctx, "SPNVMeDataType", &data); err != nil {
		slog.DebugContext(ctx, "enrich storage from nvme: system_profiler failed", "error", err)
		return
	}

	// Collect all NVMe items into a flat slice for matching.
	var nvmeItems []spNVMeItem
	for _, controller := range data.SPNVMeDataType {
		nvmeItems = append(nvmeItems, controller.Items...)
	}
	if len(nvmeItems) == 0 {
		return
	}

	for i := range devices {
		item, ok := matchNVMeItem(devices[i], nvmeItems)
		if !ok {
			continue
		}
		if devices[i].Serial == "" && item.DeviceSerial != "" {
			devices[i].Serial = item.DeviceSerial
		}
		if devices[i].FirmwareVersion == "" && item.DeviceRevision != "" {
			devices[i].FirmwareVersion = strings.TrimSpace(item.DeviceRevision)
		}
		if devices[i].SmartStatus == "" && item.SmartStatus != "" {
			if strings.EqualFold(item.SmartStatus, "Verified") {
				devices[i].SmartStatus = "PASSED"
			} else {
				devices[i].SmartStatus = item.SmartStatus
			}
		}
	}
}

// matchNVMeItem finds the NVMe item that corresponds to a StorageDevice.
// It matches by BSD device name first (most reliable), then falls back to
// a case-insensitive model name comparison.
func matchNVMeItem(dev StorageDevice, items []spNVMeItem) (spNVMeItem, bool) {
	// Extract base device name (e.g. "disk0") from the StorageDevice.
	devBSD := strings.TrimPrefix(dev.Name, "/dev/")

	// First pass: match by BSD name.
	for _, item := range items {
		if item.BSDName != "" && item.BSDName == devBSD {
			return item, true
		}
	}

	// Second pass: match by model name (case-insensitive).
	for _, item := range items {
		if item.DeviceModel != "" && strings.EqualFold(item.DeviceModel, dev.Model) {
			return item, true
		}
	}

	return spNVMeItem{}, false
}

// parseDiskutilInfo extracts Media Name, Serial Number, and SMART Status from
// `diskutil info` output. It recognises multiple field name variants that macOS
// versions use (e.g. "Device / Media Name" vs "Media Name", "Disk Serial Number"
// vs "Serial Number").
func parseDiskutilInfo(data []byte) (model, serial, smartStatus string) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if k, v, ok := strings.Cut(line, ":"); ok {
			key := strings.TrimSpace(k)
			val := strings.TrimSpace(v)
			switch key {
			case "Media Name", "Device / Media Name":
				if model == "" {
					model = val
				}
			case "Serial Number", "Disk Serial Number":
				if serial == "" {
					serial = val
				}
			case "SMART Status", "S.M.A.R.T. Status":
				if strings.EqualFold(val, "Verified") {
					smartStatus = "PASSED"
				} else if val != "" {
					smartStatus = val
				}
			}
		}
	}
	return model, serial, smartStatus
}

// parseSPStorageDataType converts system_profiler SPStorageDataType JSON into
// StorageDevice slices, grouping volumes by physical drive name. The context is
// used to run `df -P` for partition usage percentages.
func parseSPStorageDataType(ctx context.Context, data spStorageData) []StorageDevice {
	// Group volumes by physical drive device_name.
	type driveInfo struct {
		dev        StorageDevice
		partitions []PartitionInfo
	}
	driveMap := make(map[string]*driveInfo)
	var driveOrder []string

	for _, vol := range data.SPStorageDataType {
		driveName := vol.PhysicalDrive.DeviceName
		if driveName == "" {
			driveName = vol.BSDName
		}

		di, ok := driveMap[driveName]
		if !ok {
			di = &driveInfo{
				dev: StorageDevice{
					Name:      driveName,
					Model:     driveName,
					Transport: vol.PhysicalDrive.Protocol,
					Type:      classifyDarwinDiskType(vol.PhysicalDrive.MediumType, vol.PhysicalDrive.Protocol),
				},
			}
			driveMap[driveName] = di
			driveOrder = append(driveOrder, driveName)
		}

		var sz uint64
		if n, err := vol.SizeBytes.Int64(); err == nil {
			sz = uint64(n)
		}

		// Use the largest volume size as the drive size (approximation).
		if sz > di.dev.SizeBytes {
			di.dev.SizeBytes = sz
		}

		part := PartitionInfo{
			Name:       vol.BSDName,
			FSType:     vol.FileSystem,
			MountPoint: vol.MountPoint,
			SizeBytes:  sz,
		}
		if part.MountPoint != "" {
			part.UsagePct = mountUsagePctDarwin(ctx, part.MountPoint)
		}
		di.partitions = append(di.partitions, part)
	}

	var devices []StorageDevice
	for _, name := range driveOrder {
		di := driveMap[name]
		di.dev.Partitions = di.partitions
		devices = append(devices, di.dev)
	}
	return devices
}

// classifyDarwinDiskType determines if a disk is nvme, ssd, or hdd based
// on system_profiler medium_type and protocol fields.
func classifyDarwinDiskType(mediumType, protocol string) string {
	lower := strings.ToLower(mediumType + " " + protocol)
	if strings.Contains(lower, "nvme") || strings.Contains(lower, "apple fabric") {
		return "nvme"
	}
	if strings.Contains(lower, "solid state") || strings.Contains(lower, "ssd") {
		return "ssd"
	}
	if strings.Contains(lower, "rotational") || strings.Contains(lower, "hdd") {
		return "hdd"
	}
	return "ssd" // default for modern Macs
}

// ---------------------------------------------------------------------------
// GPU
// ---------------------------------------------------------------------------

type spDisplaysData struct {
	SPDisplaysDataType []spDisplayEntry `json:"SPDisplaysDataType"`
}

type spDisplayEntry struct {
	Name       string `json:"_name"`
	Model      string `json:"sppci_model"`
	VRAM       string `json:"spdisplays_vram"`
	VRAMShared string `json:"spdisplays_vram_shared"`
	PCISlot    string `json:"sppci_slot"`
}

func collectDarwinGPU(ctx context.Context) ([]GPUInfo, error) {
	var data spDisplaysData
	if err := runSystemProfiler(ctx, "SPDisplaysDataType", &data); err != nil {
		return nil, fmt.Errorf("collect gpu: %w", err)
	}

	gpus := parseSPDisplaysDataType(data)

	// Bug 4: Base M4 (and some other Apple Silicon chips) don't emit
	// spdisplays_vram or spdisplays_vram_shared. Since Apple Silicon uses
	// unified memory, fall back to total system memory as shared VRAM.
	for i := range gpus {
		if gpus[i].VRAMMB == 0 && strings.Contains(gpus[i].Model, "Apple") {
			if memBytes, err := sysctlUint64(ctx, "hw.memsize"); err == nil && memBytes > 0 {
				gpus[i].VRAMMB = int(memBytes / (1024 * 1024))
			}
		}
	}

	// Populate GPU utilization via IOKit.
	if usages := collectDarwinGPUUsage(ctx); len(usages) > 0 {
		for i := range gpus {
			if i < len(usages) {
				gpus[i].UsagePct = usages[i]
			}
		}
	}

	return gpus, nil
}

// collectDarwinGPUUsage reads GPU utilization percentages from IOKit via ioreg.
func collectDarwinGPUUsage(ctx context.Context) []int {
	out, err := exec.CommandContext(ctx, "ioreg", "-c", "IOAccelerator", "-s0", "-w0").Output()
	if err != nil {
		return nil
	}
	var usages []int
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "Device Utilization %") {
			continue
		}
		idx := strings.LastIndex(line, "=")
		if idx < 0 {
			continue
		}
		if pct, err := strconv.Atoi(strings.TrimSpace(line[idx+1:])); err == nil {
			usages = append(usages, pct)
		}
	}
	return usages
}

// parseSPDisplaysDataType converts system_profiler SPDisplaysDataType JSON
// into GPUInfo slices.
func parseSPDisplaysDataType(data spDisplaysData) []GPUInfo {
	var gpus []GPUInfo
	for _, entry := range data.SPDisplaysDataType {
		gpu := GPUInfo{
			Model:   entry.Model,
			PCISlot: entry.PCISlot,
		}

		// Parse VRAM from either dedicated or shared field.
		vramStr := entry.VRAM
		if vramStr == "" {
			vramStr = entry.VRAMShared
		}
		gpu.VRAMMB = parseVRAMMB(vramStr)

		gpus = append(gpus, gpu)
	}
	return gpus
}

// vramRe matches strings like "4096 MB" or "16 GB".
var vramRe = regexp.MustCompile(`(\d+)\s*(MB|GB)`)

// parseVRAMMB parses a VRAM string like "4096 MB" or "16 GB" into megabytes.
func parseVRAMMB(s string) int {
	m := vramRe.FindStringSubmatch(s)
	if len(m) < 3 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	if strings.ToUpper(m[2]) == "GB" {
		return n * 1024
	}
	return n
}

// ---------------------------------------------------------------------------
// Network
// ---------------------------------------------------------------------------

func collectDarwinNetwork(ctx context.Context) ([]NetworkInfo, error) {
	ifconfigOut, err := exec.CommandContext(ctx, "ifconfig").Output()
	if err != nil {
		return nil, fmt.Errorf("collect network: run ifconfig: %w", err)
	}

	nics := parseIfconfig(ifconfigOut)

	// Try to get port type mapping from networksetup.
	if nsOut, err := exec.CommandContext(ctx, "networksetup", "-listallhardwareports").Output(); err == nil {
		portMap := parseNetworksetupPorts(nsOut)
		for i := range nics {
			if portType, ok := portMap[nics[i].Name]; ok {
				nics[i].Type = portType
			}
		}
	}

	return nics, nil
}

// parseIfconfig parses macOS ifconfig output into NetworkInfo slices.
// Skips the lo0 (loopback) interface.
func parseIfconfig(data []byte) []NetworkInfo {
	var nics []NetworkInfo
	var current *NetworkInfo

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()

		// Interface header: "en0: flags=8863<...> mtu 1500"
		if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
			// Save previous interface.
			if current != nil && current.Name != "lo0" {
				nics = append(nics, *current)
			}

			name, _, ok := strings.Cut(line, ":")
			if !ok {
				current = nil
				continue
			}

			current = &NetworkInfo{Name: name}

			// Parse MTU from "mtu NNNN".
			if idx := strings.Index(line, "mtu "); idx >= 0 {
				mtuStr := line[idx+4:]
				if sp := strings.IndexByte(mtuStr, ' '); sp >= 0 {
					mtuStr = mtuStr[:sp]
				}
				current.MTU, _ = strconv.Atoi(strings.TrimSpace(mtuStr))
			}

			continue
		}

		if current == nil {
			continue
		}

		trimmed := strings.TrimSpace(line)

		// ether aa:bb:cc:dd:ee:ff
		if strings.HasPrefix(trimmed, "ether ") {
			current.MACAddress = strings.TrimPrefix(trimmed, "ether ")
			continue
		}

		// inet 192.168.1.100 netmask 0xffffff00 broadcast 192.168.1.255
		if strings.HasPrefix(trimmed, "inet ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 4 && parts[2] == "netmask" {
				ip := IPAddress{
					Address:   parts[1],
					PrefixLen: parseHexNetmask(parts[3]),
				}
				current.IPv4Addresses = append(current.IPv4Addresses, ip)
			}
			continue
		}

		// inet6 fe80::1 prefixlen 64 scopeid 0x6
		if strings.HasPrefix(trimmed, "inet6 ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 4 && parts[2] == "prefixlen" {
				addr := parts[1]
				// Strip %interface suffix from link-local addresses.
				if idx := strings.IndexByte(addr, '%'); idx >= 0 {
					addr = addr[:idx]
				}
				prefixLen, _ := strconv.Atoi(parts[3])
				ip := IPAddress{
					Address:   addr,
					PrefixLen: prefixLen,
				}
				current.IPv6Addresses = append(current.IPv6Addresses, ip)
			}
			continue
		}

		// status: active / inactive
		if strings.HasPrefix(trimmed, "status: ") {
			current.State = strings.TrimPrefix(trimmed, "status: ")
			continue
		}
	}

	// Don't forget the last interface.
	if current != nil && current.Name != "lo0" {
		nics = append(nics, *current)
	}

	// Filter out virtual/tunnel interfaces that provide no useful hardware info.
	return filterPhysicalNICs(nics)
}

// virtualNICPrefixes lists macOS interface name prefixes for virtual/tunnel
// interfaces that should be excluded from hardware inventory.
var virtualNICPrefixes = []string{
	"utun", "gif", "stf", "llw", "anpi", "ap", "awdl",
}

// filterPhysicalNICs removes virtual/tunnel interfaces from the list.
// An interface is kept if it has a MAC address AND (has an IPv4 address OR is active).
// Interfaces matching known virtual prefixes are always removed.
func filterPhysicalNICs(nics []NetworkInfo) []NetworkInfo {
	var filtered []NetworkInfo
	for _, nic := range nics {
		if isVirtualNIC(nic.Name) {
			continue
		}
		// Keep interfaces that have a MAC and are either active or have an IPv4 address.
		if nic.MACAddress != "" && (nic.State == "active" || len(nic.IPv4Addresses) > 0) {
			filtered = append(filtered, nic)
		}
	}
	return filtered
}

// isVirtualNIC returns true if the interface name matches a known virtual/tunnel prefix.
func isVirtualNIC(name string) bool {
	for _, prefix := range virtualNICPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// parseNetworksetupPorts parses `networksetup -listallhardwareports` output
// and returns a map of device name to port type (e.g. "en0" -> "Wi-Fi").
func parseNetworksetupPorts(data []byte) map[string]string {
	result := make(map[string]string)

	var currentPort string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "Hardware Port: ") {
			currentPort = strings.TrimPrefix(line, "Hardware Port: ")
			continue
		}

		if strings.HasPrefix(line, "Device: ") && currentPort != "" {
			device := strings.TrimPrefix(line, "Device: ")
			result[device] = currentPort
			currentPort = ""
			continue
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// USB
// ---------------------------------------------------------------------------

type spUSBData struct {
	SPUSBDataType []spUSBBus `json:"SPUSBDataType"`
}

type spUSBBus struct {
	Name           string       `json:"_name"`
	HostController string       `json:"host_controller"`
	Items          []spUSBEntry `json:"_items"`
}

type spUSBEntry struct {
	Name       string       `json:"_name"`
	LocationID string       `json:"location_id"`
	VendorID   string       `json:"vendor_id"`
	ProductID  string       `json:"product_id"`
	Items      []spUSBEntry `json:"_items"`
}

func collectDarwinUSB(ctx context.Context) ([]USBDevice, error) {
	var data spUSBData
	if err := runSystemProfiler(ctx, "SPUSBDataType", &data); err != nil {
		return nil, fmt.Errorf("collect usb: %w", err)
	}

	return parseSPUSBDataType(data), nil
}

// parseSPUSBDataType extracts USB devices from nested system_profiler
// SPUSBDataType JSON. Walks recursively through hub _items arrays.
func parseSPUSBDataType(data spUSBData) []USBDevice {
	var devices []USBDevice
	for _, bus := range data.SPUSBDataType {
		walkUSBItems(bus.Items, &devices)
	}
	return devices
}

func walkUSBItems(items []spUSBEntry, devices *[]USBDevice) {
	for _, item := range items {
		dev := USBDevice{
			Description: item.Name,
			VendorID:    item.VendorID,
			ProductID:   item.ProductID,
			DeviceNum:   item.LocationID,
		}
		*devices = append(*devices, dev)

		// Recurse into nested hubs.
		if len(item.Items) > 0 {
			walkUSBItems(item.Items, devices)
		}
	}
}

// ---------------------------------------------------------------------------
// Battery
// ---------------------------------------------------------------------------

func collectDarwinBattery(ctx context.Context) (*BatteryInfo, error) {
	out, err := exec.CommandContext(ctx, "ioreg", "-r", "-c", "AppleSmartBattery").Output()
	if err != nil {
		return &BatteryInfo{Present: false}, nil
	}

	return parseIoregBattery(out), nil
}

// ioregKVRe matches key-value pairs in ioreg output like: "CycleCount" = 342
var ioregKVRe = regexp.MustCompile(`"(\w+)"\s*=\s*(.+)`)

// parseIoregBattery parses `ioreg -r -c AppleSmartBattery` text output into BatteryInfo.
func parseIoregBattery(data []byte) *BatteryInfo {
	info := &BatteryInfo{}

	s := strings.TrimSpace(string(data))
	if s == "" {
		return info
	}

	vals := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		m := ioregKVRe.FindStringSubmatch(scanner.Text())
		if len(m) < 3 {
			continue
		}
		key := m[1]
		val := strings.TrimSpace(m[2])
		vals[key] = val
	}

	if vals["BatteryInstalled"] != "Yes" {
		return info
	}
	info.Present = true
	info.Technology = "lithium-ion"

	cycleCount, _ := strconv.Atoi(vals["CycleCount"])
	info.CycleCount = cycleCount

	// On macOS 15+, MaxCapacity reports a percentage and the raw mAh value
	// moves to AppleRawMaxCapacity. Same for CurrentCapacity / AppleRawCurrentCapacity.
	rawMaxCap, _ := strconv.Atoi(vals["AppleRawMaxCapacity"])
	maxCap, _ := strconv.Atoi(vals["MaxCapacity"])
	rawCurCap, _ := strconv.Atoi(vals["AppleRawCurrentCapacity"])
	currentCap, _ := strconv.Atoi(vals["CurrentCapacity"])
	designCap, _ := strconv.Atoi(vals["DesignCapacity"])

	effectiveMax := rawMaxCap
	if effectiveMax == 0 {
		effectiveMax = maxCap
	}
	effectiveCur := rawCurCap
	if effectiveCur == 0 {
		effectiveCur = currentCap
	}

	if effectiveMax > 0 {
		info.CapacityPct = effectiveCur * 100 / effectiveMax
	}

	if designCap > 0 {
		info.HealthPct = effectiveMax * 100 / designCap
		// Clamp to 100%: new batteries can report MaxCapacity > DesignCapacity.
		if info.HealthPct > 100 {
			info.HealthPct = 100
		}
	}

	// Determine status.
	switch {
	case vals["IsCharging"] == "Yes":
		info.Status = "Charging"
	case vals["FullyCharged"] == "Yes":
		info.Status = "Full"
	default:
		info.Status = "Discharging"
	}

	return info
}

// ---------------------------------------------------------------------------
// Virtualization
// ---------------------------------------------------------------------------

func collectDarwinVirtualization(ctx context.Context) (*VirtInfo, error) {
	model, err := sysctlString(ctx, "hw.model")
	if err != nil {
		return &VirtInfo{}, nil
	}
	return detectVirtFromModel(model), nil
}

// detectVirtFromModel checks the hw.model string for known hypervisor identifiers.
func detectVirtFromModel(model string) *VirtInfo {
	lower := strings.ToLower(model)
	hypervisors := []struct {
		substring  string
		hypervisor string
	}{
		{"vmware", "vmware"},
		{"virtualbox", "virtualbox"},
		{"parallels", "parallels"},
		{"qemu", "qemu"},
	}

	for _, h := range hypervisors {
		if strings.Contains(lower, h.substring) {
			return &VirtInfo{
				IsVirtual:      true,
				HypervisorType: h.hypervisor,
			}
		}
	}

	return &VirtInfo{}
}
