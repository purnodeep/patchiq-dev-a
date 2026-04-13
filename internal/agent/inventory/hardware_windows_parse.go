package inventory

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// winArchString converts a Win32_Processor Architecture integer to a string.
func winArchString(arch int) string {
	switch arch {
	case 9:
		return "x86_64"
	case 12:
		return "ARM64"
	case 0:
		return "x86"
	default:
		return strconv.Itoa(arch)
	}
}

// winMemTypeString converts SMBIOSMemoryType to a human-readable string.
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

// winFormFactorString converts FormFactor integer to a human-readable string.
func winFormFactorString(f int) string {
	switch f {
	case 8:
		return "DIMM"
	case 12:
		return "SODIMM"
	case 13:
		return "TSOP"
	default:
		return "Unknown"
	}
}

// winDiskType maps MediaType and InterfaceType to a normalized disk type string.
func winDiskType(mediaType, interfaceType string) string {
	lower := strings.ToLower(mediaType)
	iface := strings.ToLower(interfaceType)
	if iface == "nvme" || strings.Contains(lower, "nvme") {
		return "nvme"
	}
	if strings.Contains(lower, "solid state") || strings.Contains(lower, "ssd") {
		return "ssd"
	}
	if strings.Contains(lower, "fixed hard disk") || strings.Contains(lower, "hdd") {
		return "hdd"
	}
	return "unknown"
}

// parseWinLinkSpeed parses a Windows link speed string like "1 Gbps" or "100 Mbps" to Mbps.
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
	default:
		return 0
	}
}

// classifyWinNetType infers the network type from the interface description.
func classifyWinNetType(desc string) string {
	lower := strings.ToLower(desc)
	if strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wifi") || strings.Contains(lower, "wireless") || strings.Contains(lower, "802.11") {
		return "wifi"
	}
	if strings.Contains(lower, "virtual") || strings.Contains(lower, "vmware") || strings.Contains(lower, "hyper-v") || strings.Contains(lower, "loopback") || strings.Contains(lower, "tap") || strings.Contains(lower, "tunnel") {
		return "virtual"
	}
	return "ethernet"
}

var usbVIDRe = regexp.MustCompile(`(?i)VID_([0-9A-F]{4})`)
var usbPIDRe = regexp.MustCompile(`(?i)PID_([0-9A-F]{4})`)

// extractUSBIDs extracts VendorID and ProductID from a PNPDeviceID string.
func extractUSBIDs(pnpID string) (string, string) {
	vid := ""
	pid := ""
	if m := usbVIDRe.FindStringSubmatch(pnpID); len(m) == 2 {
		vid = strings.ToLower(m[1])
	}
	if m := usbPIDRe.FindStringSubmatch(pnpID); len(m) == 2 {
		pid = strings.ToLower(m[1])
	}
	return vid, pid
}

// winBatteryStatusString converts BatteryStatus integer to a human-readable string.
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

// winChemistryString converts Win32_Battery Chemistry integer to a string.
func winChemistryString(c int) string {
	switch c {
	case 3:
		return "Lead Acid"
	case 4:
		return "Nickel Cadmium"
	case 5:
		return "Nickel Metal Hydride"
	case 6:
		return "Li-ion"
	case 7:
		return "Zinc air"
	case 8:
		return "Lithium Polymer"
	default:
		return "Unknown"
	}
}

// normalizeJSON ensures the input is a JSON array. PowerShell sometimes returns
// a single object instead of a one-element array.
func normalizeJSON(data string) string {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return "[]"
	}
	if data[0] == '[' {
		return data
	}
	return "[" + data + "]"
}

// parseWinCPU parses Win32_Processor JSON into CPUInfo.
func parseWinCPU(data string) CPUInfo {
	type winProc struct {
		Name                          string  `json:"Name"`
		Manufacturer                  string  `json:"Manufacturer"`
		Family                        int     `json:"Family"`
		NumberOfCores                 int     `json:"NumberOfCores"`
		NumberOfLogicalProcessors     int     `json:"NumberOfLogicalProcessors"`
		MaxClockSpeed                 float64 `json:"MaxClockSpeed"`
		L2CacheSize                   int     `json:"L2CacheSize"`
		L3CacheSize                   int     `json:"L3CacheSize"`
		Architecture                  int     `json:"Architecture"`
		VirtualizationFirmwareEnabled bool    `json:"VirtualizationFirmwareEnabled"`
	}

	var procs []winProc
	if err := json.Unmarshal([]byte(normalizeJSON(data)), &procs); err != nil || len(procs) == 0 {
		return CPUInfo{}
	}

	p := procs[0]
	threadsPerCore := 0
	if p.NumberOfCores > 0 {
		threadsPerCore = p.NumberOfLogicalProcessors / p.NumberOfCores
	}

	virtType := ""
	if p.VirtualizationFirmwareEnabled {
		virtType = "hyper-v"
	}

	return CPUInfo{
		ModelName:      p.Name,
		Vendor:         p.Manufacturer,
		Family:         strconv.Itoa(p.Family),
		CoresPerSocket: p.NumberOfCores,
		ThreadsPerCore: threadsPerCore,
		Sockets:        len(procs),
		TotalLogical:   p.NumberOfLogicalProcessors,
		MaxMHz:         p.MaxClockSpeed,
		CacheL2:        fmt.Sprintf("%d KiB", p.L2CacheSize),
		CacheL3:        fmt.Sprintf("%d KiB", p.L3CacheSize),
		Architecture:   winArchString(p.Architecture),
		VirtType:       virtType,
	}
}

// parseWinMemory parses Win32_OperatingSystem and Win32_PhysicalMemory JSON into MemoryInfo.
func parseWinMemory(osData, dimmData string) MemoryInfo {
	type winOS struct {
		TotalVisibleMemorySize uint64 `json:"TotalVisibleMemorySize"`
		FreePhysicalMemory     uint64 `json:"FreePhysicalMemory"`
	}

	type winDIMM struct {
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

	var osInfo winOS
	_ = json.Unmarshal([]byte(osData), &osInfo)

	var dimms []winDIMM
	_ = json.Unmarshal([]byte(normalizeJSON(dimmData)), &dimms)

	info := MemoryInfo{
		TotalBytes:     osInfo.TotalVisibleMemorySize * 1024,
		AvailableBytes: osInfo.FreePhysicalMemory * 1024,
		NumSlots:       len(dimms),
	}

	for _, d := range dimms {
		info.DIMMs = append(info.DIMMs, DIMMInfo{
			Locator:      d.DeviceLocator,
			BankLocator:  d.BankLabel,
			SizeMB:       int(d.Capacity / 1048576),
			Type:         winMemTypeString(d.SMBIOSMemoryType),
			SpeedMHz:     d.ConfiguredClockSpeed,
			Manufacturer: d.Manufacturer,
			SerialNumber: d.SerialNumber,
			PartNumber:   d.PartNumber,
			FormFactor:   winFormFactorString(d.FormFactor),
		})
	}

	return info
}

// parseWinMotherboard parses a combined board+bios JSON into MotherboardInfo.
func parseWinMotherboard(data string) MotherboardInfo {
	type winBoard struct {
		Manufacturer string `json:"Manufacturer"`
		Product      string `json:"Product"`
		Version      string `json:"Version"`
		SerialNumber string `json:"SerialNumber"`
	}
	type winBIOS struct {
		Manufacturer      string `json:"Manufacturer"`
		SMBIOSBIOSVersion string `json:"SMBIOSBIOSVersion"`
		ReleaseDate       string `json:"ReleaseDate"`
	}
	type winMB struct {
		Board winBoard `json:"board"`
		BIOS  winBIOS  `json:"bios"`
	}

	var mb winMB
	if err := json.Unmarshal([]byte(data), &mb); err != nil {
		return MotherboardInfo{}
	}

	releaseDate := mb.BIOS.ReleaseDate
	if idx := strings.Index(releaseDate, "T"); idx >= 0 {
		releaseDate = releaseDate[:idx]
	}

	return MotherboardInfo{
		BoardManufacturer: mb.Board.Manufacturer,
		BoardProduct:      mb.Board.Product,
		BoardVersion:      mb.Board.Version,
		BoardSerial:       mb.Board.SerialNumber,
		BIOSVendor:        mb.BIOS.Manufacturer,
		BIOSVersion:       mb.BIOS.SMBIOSBIOSVersion,
		BIOSReleaseDate:   releaseDate,
	}
}

// parseWinStorage parses Win32_DiskDrive and Win32_LogicalDisk JSON into []StorageDevice.
func parseWinStorage(diskData, logicalData string) []StorageDevice {
	type winDisk struct {
		DeviceID         string `json:"DeviceID"`
		Model            string `json:"Model"`
		SerialNumber     string `json:"SerialNumber"`
		Size             uint64 `json:"Size"`
		MediaType        string `json:"MediaType"`
		InterfaceType    string `json:"InterfaceType"`
		FirmwareRevision string `json:"FirmwareRevision"`
		Status           string `json:"Status"`
		Partitions       int    `json:"Partitions"`
	}

	type winLogical struct {
		DeviceID   string `json:"DeviceID"`
		Size       uint64 `json:"Size"`
		FreeSpace  uint64 `json:"FreeSpace"`
		FileSystem string `json:"FileSystem"`
		VolumeName string `json:"VolumeName"`
	}

	var disks []winDisk
	_ = json.Unmarshal([]byte(normalizeJSON(diskData)), &disks)

	var logicals []winLogical
	_ = json.Unmarshal([]byte(normalizeJSON(logicalData)), &logicals)

	// Build all partitions
	var partitions []PartitionInfo
	for _, l := range logicals {
		usagePct := 0
		if l.Size > 0 {
			usagePct = int((l.Size - l.FreeSpace) * 100 / l.Size)
		}
		partitions = append(partitions, PartitionInfo{
			Name:      l.DeviceID,
			SizeBytes: l.Size,
			FSType:    l.FileSystem,
			UsagePct:  usagePct,
		})
	}

	var result []StorageDevice
	for _, d := range disks {
		smartStatus := ""
		if strings.EqualFold(d.Status, "OK") {
			smartStatus = "PASSED"
		}
		sd := StorageDevice{
			Name:            d.DeviceID,
			Model:           d.Model,
			Serial:          strings.TrimSpace(d.SerialNumber),
			SizeBytes:       d.Size,
			Type:            winDiskType(d.MediaType, d.InterfaceType),
			FirmwareVersion: d.FirmwareRevision,
			Transport:       d.InterfaceType,
			SmartStatus:     smartStatus,
			Partitions:      partitions,
		}
		result = append(result, sd)
	}

	return result
}

// parseWinGPU parses Win32_VideoController JSON into []GPUInfo.
// usageStr is the total 3D engine utilization percentage as a string (may be empty).
func parseWinGPU(data, usageStr string) []GPUInfo {
	type winGPU struct {
		Name          string `json:"Name"`
		AdapterRAM    uint64 `json:"AdapterRAM"`
		DriverVersion string `json:"DriverVersion"`
		PNPDeviceID   string `json:"PNPDeviceID"`
	}

	var gpus []winGPU
	if err := json.Unmarshal([]byte(normalizeJSON(data)), &gpus); err != nil {
		return nil
	}

	usagePct, _ := strconv.Atoi(strings.TrimSpace(usageStr))

	var result []GPUInfo
	for i, g := range gpus {
		pciSlot := ""
		if idx := strings.Index(strings.ToUpper(g.PNPDeviceID), "PCI\\"); idx >= 0 {
			part := g.PNPDeviceID[idx:]
			if end := strings.Index(part[4:], "\\"); end >= 0 {
				pciSlot = part[:end+4]
			} else {
				pciSlot = part
			}
		}
		gpu := GPUInfo{
			Model:         g.Name,
			VRAMMB:        int(g.AdapterRAM / 1048576),
			DriverVersion: g.DriverVersion,
			PCISlot:       pciSlot,
		}
		// Assign total 3D utilization to the first GPU only.
		if i == 0 {
			gpu.UsagePct = usagePct
		}
		result = append(result, gpu)
	}

	return result
}

// parseWinNetwork parses a JSON object with "adapters" and "ips" arrays into []NetworkInfo.
func parseWinNetwork(data string) []NetworkInfo {
	type winAdapter struct {
		Name                 string `json:"Name"`
		MacAddress           string `json:"MacAddress"`
		MtuSize              int    `json:"MtuSize"`
		Status               string `json:"Status"`
		LinkSpeed            string `json:"LinkSpeed"`
		InterfaceDescription string `json:"InterfaceDescription"`
		DriverName           string `json:"DriverName"`
	}
	type winIP struct {
		InterfaceAlias string `json:"InterfaceAlias"`
		IPAddress      string `json:"IPAddress"`
		PrefixLength   int    `json:"PrefixLength"`
		AddressFamily  int    `json:"AddressFamily"`
	}
	type winNet struct {
		Adapters []winAdapter `json:"adapters"`
		IPs      []winIP      `json:"ips"`
	}

	var wn winNet
	if err := json.Unmarshal([]byte(data), &wn); err != nil {
		return nil
	}

	var result []NetworkInfo
	for _, a := range wn.Adapters {
		mac := strings.ReplaceAll(a.MacAddress, "-", ":")
		ni := NetworkInfo{
			Name:       a.Name,
			MACAddress: mac,
			MTU:        a.MtuSize,
			State:      strings.ToLower(a.Status),
			SpeedMbps:  parseWinLinkSpeed(a.LinkSpeed),
			Type:       classifyWinNetType(a.InterfaceDescription),
			Driver:     a.DriverName,
		}
		for _, ip := range wn.IPs {
			if ip.InterfaceAlias != a.Name {
				continue
			}
			addr := IPAddress{Address: ip.IPAddress, PrefixLen: ip.PrefixLength}
			switch ip.AddressFamily {
			case 2:
				ni.IPv4Addresses = append(ni.IPv4Addresses, addr)
			case 23:
				ni.IPv6Addresses = append(ni.IPv6Addresses, addr)
			}
		}
		result = append(result, ni)
	}

	return result
}

// parseWinUSB parses Win32_PnPEntity JSON into []USBDevice.
func parseWinUSB(data string) []USBDevice {
	type winPnP struct {
		PNPDeviceID string `json:"PNPDeviceID"`
		Name        string `json:"Name"`
	}

	var entries []winPnP
	if err := json.Unmarshal([]byte(normalizeJSON(data)), &entries); err != nil {
		return nil
	}

	var result []USBDevice
	for _, e := range entries {
		vid, pid := extractUSBIDs(e.PNPDeviceID)
		result = append(result, USBDevice{
			Bus:         "USB",
			VendorID:    vid,
			ProductID:   pid,
			Description: e.Name,
		})
	}

	return result
}

// parseWinBattery parses Win32_Battery JSON into BatteryInfo.
func parseWinBattery(data string) BatteryInfo {
	type winBat struct {
		BatteryStatus            int `json:"BatteryStatus"`
		EstimatedChargeRemaining int `json:"EstimatedChargeRemaining"`
		DesignCapacity           int `json:"DesignCapacity"`
		FullChargeCapacity       int `json:"FullChargeCapacity"`
		Chemistry                int `json:"Chemistry"`
	}

	var bats []winBat
	if err := json.Unmarshal([]byte(normalizeJSON(data)), &bats); err != nil || len(bats) == 0 {
		return BatteryInfo{Present: false}
	}

	b := bats[0]
	healthPct := 0
	if b.DesignCapacity > 0 {
		healthPct = int(float64(b.FullChargeCapacity) / float64(b.DesignCapacity) * 100)
	}

	return BatteryInfo{
		Present:        true,
		Status:         winBatteryStatusString(b.BatteryStatus),
		CapacityPct:    b.EstimatedChargeRemaining,
		EnergyFullWh:   float64(b.FullChargeCapacity) / 1000,
		EnergyDesignWh: float64(b.DesignCapacity) / 1000,
		HealthPct:      healthPct,
		Technology:     winChemistryString(b.Chemistry),
	}
}

// parseWinTPM parses Get-Tpm JSON output into TPMInfo.
func parseWinTPM(data string) TPMInfo {
	type winTPM struct {
		TpmPresent          bool   `json:"TpmPresent"`
		ManufacturerVersion string `json:"ManufacturerVersion"`
	}

	var t winTPM
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return TPMInfo{}
	}

	return TPMInfo{
		Present: t.TpmPresent,
		Version: t.ManufacturerVersion,
	}
}

// parseWinVirtualization parses Win32_ComputerSystem JSON into VirtInfo.
func parseWinVirtualization(data string) VirtInfo {
	type winCS struct {
		Model             string `json:"Model"`
		HypervisorPresent bool   `json:"HypervisorPresent"`
	}

	var cs winCS
	if err := json.Unmarshal([]byte(data), &cs); err != nil {
		return VirtInfo{}
	}

	modelLower := strings.ToLower(cs.Model)
	isVirtual := cs.HypervisorPresent
	hypervisorType := ""

	if strings.Contains(modelLower, "vmware") {
		isVirtual = true
		hypervisorType = "vmware"
	} else if strings.Contains(modelLower, "virtualbox") {
		isVirtual = true
		hypervisorType = "virtualbox"
	} else if strings.Contains(modelLower, "virtual machine") {
		isVirtual = true
		hypervisorType = "hyper-v"
	} else if strings.Contains(modelLower, "qemu") || strings.Contains(modelLower, "kvm") {
		isVirtual = true
		hypervisorType = "kvm"
	} else if cs.HypervisorPresent {
		hypervisorType = "hyper-v"
	}

	return VirtInfo{
		IsVirtual:      isVirtual,
		HypervisorType: hypervisorType,
	}
}
