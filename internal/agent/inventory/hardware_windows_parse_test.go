package inventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "windows", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(data)
}

func TestParseWinCPU(t *testing.T) {
	cpu := parseWinCPU(readFixture(t, "cpu.json"))
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
		t.Errorf("MaxMHz = %v, want 5400", cpu.MaxMHz)
	}
	if cpu.Architecture != "x86_64" {
		t.Errorf("Architecture = %q, want x86_64", cpu.Architecture)
	}
	if cpu.CacheL2 != "24576 KiB" {
		t.Errorf("CacheL2 = %q, want '24576 KiB'", cpu.CacheL2)
	}
	if cpu.VirtType != "hyper-v" {
		t.Errorf("VirtType = %q, want 'hyper-v'", cpu.VirtType)
	}
	if cpu.Sockets != 1 {
		t.Errorf("Sockets = %d, want 1", cpu.Sockets)
	}
}

func TestParseWinCPUSingleObject(t *testing.T) {
	// PowerShell may return a single object, not an array
	data := `{"Name":"Intel Xeon","Manufacturer":"GenuineIntel","NumberOfCores":4,"NumberOfLogicalProcessors":8,"MaxClockSpeed":3000,"Architecture":9}`
	cpu := parseWinCPU(data)
	if cpu.ModelName != "Intel Xeon" {
		t.Errorf("ModelName = %q, want 'Intel Xeon'", cpu.ModelName)
	}
	if cpu.Sockets != 1 {
		t.Errorf("Sockets = %d, want 1", cpu.Sockets)
	}
}

func TestParseWinCPUEmpty(t *testing.T) {
	cpu := parseWinCPU("")
	if cpu.ModelName != "" {
		t.Errorf("expected empty CPUInfo for empty input")
	}
}

func TestParseWinMemory(t *testing.T) {
	mem := parseWinMemory(readFixture(t, "memory_os.json"), readFixture(t, "memory_dimm.json"))
	wantTotal := uint64(16777216 * 1024)
	if mem.TotalBytes != wantTotal {
		t.Errorf("TotalBytes = %d, want %d", mem.TotalBytes, wantTotal)
	}
	wantAvail := uint64(8388608 * 1024)
	if mem.AvailableBytes != wantAvail {
		t.Errorf("AvailableBytes = %d, want %d", mem.AvailableBytes, wantAvail)
	}
	if mem.NumSlots != 1 {
		t.Errorf("NumSlots = %d, want 1", mem.NumSlots)
	}
	if len(mem.DIMMs) != 1 {
		t.Fatalf("len(DIMMs) = %d, want 1", len(mem.DIMMs))
	}
	d := mem.DIMMs[0]
	if d.SizeMB != 16384 {
		t.Errorf("DIMM SizeMB = %d, want 16384", d.SizeMB)
	}
	if d.Type != "DDR5" {
		t.Errorf("DIMM Type = %q, want DDR5", d.Type)
	}
	if d.SpeedMHz != 4800 {
		t.Errorf("DIMM SpeedMHz = %d, want 4800", d.SpeedMHz)
	}
	if d.FormFactor != "SODIMM" {
		t.Errorf("DIMM FormFactor = %q, want SODIMM", d.FormFactor)
	}
}

func TestParseWinMotherboard(t *testing.T) {
	mb := parseWinMotherboard(readFixture(t, "motherboard.json"))
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
		t.Errorf("BIOSVersion = %q, want '2803'", mb.BIOSVersion)
	}
	if mb.BIOSReleaseDate != "2024-01-15" {
		t.Errorf("BIOSReleaseDate = %q, want '2024-01-15'", mb.BIOSReleaseDate)
	}
}

func TestParseWinStorage(t *testing.T) {
	devices := parseWinStorage(readFixture(t, "disks.json"), readFixture(t, "logical_disks.json"))
	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}
	d := devices[0]
	if !strings.Contains(d.Model, "Samsung") {
		t.Errorf("Model = %q, should contain 'Samsung'", d.Model)
	}
	if d.Type != "nvme" {
		t.Errorf("Type = %q, want 'nvme'", d.Type)
	}
	if d.SmartStatus != "PASSED" {
		t.Errorf("SmartStatus = %q, want 'PASSED'", d.SmartStatus)
	}
	if len(d.Partitions) != 2 {
		t.Errorf("len(Partitions) = %d, want 2", len(d.Partitions))
	}
	if d.Partitions[0].FSType != "NTFS" {
		t.Errorf("FSType = %q, want 'NTFS'", d.Partitions[0].FSType)
	}
}

func TestParseWinGPU(t *testing.T) {
	gpus := parseWinGPU(readFixture(t, "gpu.json"), "")
	if len(gpus) != 1 {
		t.Fatalf("len(gpus) = %d, want 1", len(gpus))
	}
	g := gpus[0]
	if g.Model != "NVIDIA GeForce RTX 4090" {
		t.Errorf("Model = %q", g.Model)
	}
	if g.VRAMMB != 24576 {
		t.Errorf("VRAMMB = %d, want 24576", g.VRAMMB)
	}
}

func TestParseWinNetwork(t *testing.T) {
	nics := parseWinNetwork(readFixture(t, "network.json"))
	if len(nics) != 1 {
		t.Fatalf("len(nics) = %d, want 1", len(nics))
	}
	n := nics[0]
	if n.MACAddress != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("MACAddress = %q, want 'AA:BB:CC:DD:EE:FF'", n.MACAddress)
	}
	if n.SpeedMbps != 1000 {
		t.Errorf("SpeedMbps = %d, want 1000", n.SpeedMbps)
	}
	if len(n.IPv4Addresses) != 1 {
		t.Errorf("IPv4 count = %d, want 1", len(n.IPv4Addresses))
	}
	if len(n.IPv6Addresses) != 1 {
		t.Errorf("IPv6 count = %d, want 1", len(n.IPv6Addresses))
	}
}

func TestParseWinUSB(t *testing.T) {
	devs := parseWinUSB(readFixture(t, "usb.json"))
	if len(devs) != 2 {
		t.Fatalf("len(devs) = %d, want 2", len(devs))
	}
	d := devs[0]
	if d.VendorID != "046d" {
		t.Errorf("VendorID = %q, want '046d'", d.VendorID)
	}
	if d.ProductID != "c52b" {
		t.Errorf("ProductID = %q, want 'c52b'", d.ProductID)
	}
}

func TestParseWinBattery(t *testing.T) {
	bat := parseWinBattery(readFixture(t, "battery.json"))
	if !bat.Present {
		t.Error("Present should be true")
	}
	if bat.CapacityPct != 85 {
		t.Errorf("CapacityPct = %d, want 85", bat.CapacityPct)
	}
	if bat.HealthPct != 95 {
		t.Errorf("HealthPct = %d, want 95", bat.HealthPct)
	}
	if bat.Status != "AC" {
		t.Errorf("Status = %q, want 'AC'", bat.Status)
	}
	if bat.Technology != "Unknown" {
		t.Errorf("Technology = %q, want 'Unknown'", bat.Technology)
	}
}

func TestParseWinBatteryEmpty(t *testing.T) {
	bat := parseWinBattery("[]")
	if bat.Present {
		t.Error("Present should be false for empty array")
	}
}

func TestParseWinTPM(t *testing.T) {
	tpm := parseWinTPM(readFixture(t, "tpm.json"))
	if !tpm.Present {
		t.Error("Present should be true")
	}
	if tpm.Version != "2.0" {
		t.Errorf("Version = %q, want '2.0'", tpm.Version)
	}
}

func TestParseWinVirtualizationBareMetal(t *testing.T) {
	virt := parseWinVirtualization(readFixture(t, "computer_system.json"))
	if virt.IsVirtual {
		t.Error("IsVirtual should be false for bare metal")
	}
}

func TestParseWinVirtualizationVM(t *testing.T) {
	virt := parseWinVirtualization(readFixture(t, "computer_system_vm.json"))
	if !virt.IsVirtual {
		t.Error("IsVirtual should be true for VM")
	}
	if virt.HypervisorType != "hyper-v" {
		t.Errorf("HypervisorType = %q, want 'hyper-v'", virt.HypervisorType)
	}
}

func TestExtractUSBIDs(t *testing.T) {
	vid, pid := extractUSBIDs("USB\\VID_046D&PID_C52B\\6&ABC123")
	if vid != "046d" {
		t.Errorf("vid = %q, want '046d'", vid)
	}
	if pid != "c52b" {
		t.Errorf("pid = %q, want 'c52b'", pid)
	}

	// No IDs
	vid2, pid2 := extractUSBIDs("ACPI\\PNP0303\\4&1A2B3C4")
	if vid2 != "" || pid2 != "" {
		t.Errorf("expected empty IDs, got vid=%q pid=%q", vid2, pid2)
	}
}

func TestParseWinLinkSpeed(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1 Gbps", 1000},
		{"100 Mbps", 100},
		{"10 Gbps", 10000},
		{"", 0},
		{"unknown", 0},
	}
	for _, tc := range tests {
		got := parseWinLinkSpeed(tc.input)
		if got != tc.want {
			t.Errorf("parseWinLinkSpeed(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
