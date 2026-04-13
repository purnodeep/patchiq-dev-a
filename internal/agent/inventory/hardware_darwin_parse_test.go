//go:build darwin

package inventory

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestFormatCacheSize(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, ""},
		{512, "512 B"},
		{1024, "1 KiB"},
		{32768, "32 KiB"},
		{65536, "64 KiB"},
		{131072, "128 KiB"},
		{1048576, "1 MiB"},
		{4194304, "4 MiB"},
		{16777216, "16 MiB"},
		{33554432, "32 MiB"},
		{1073741824, "1 GiB"},
	}

	for _, tt := range tests {
		got := formatCacheSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatCacheSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestParseHexNetmask(t *testing.T) {
	tests := []struct {
		hex  string
		want int
	}{
		{"0xffffff00", 24},
		{"0xffff0000", 16},
		{"0xffffffff", 32},
		{"0xff000000", 8},
		{"0xfffff800", 21},
		{"0xfffffffe", 31},
		{"0x00000000", 0},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		got := parseHexNetmask(tt.hex)
		if got != tt.want {
			t.Errorf("parseHexNetmask(%q) = %d, want %d", tt.hex, got, tt.want)
		}
	}
}

func TestParseIfconfig(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/ifconfig.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	nics := parseIfconfig(data)

	// lo0, utun0 (virtual) excluded; en1 (inactive, no IPv4) filtered; expecting en0, en2.
	if len(nics) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(nics))
	}

	// en0: Wi-Fi with IPv4 and IPv6.
	en0 := nics[0]
	if en0.Name != "en0" {
		t.Errorf("nics[0].Name = %q, want %q", en0.Name, "en0")
	}
	if en0.MTU != 1500 {
		t.Errorf("en0.MTU = %d, want 1500", en0.MTU)
	}
	if en0.MACAddress != "a4:83:e7:2b:1c:d0" {
		t.Errorf("en0.MACAddress = %q, want %q", en0.MACAddress, "a4:83:e7:2b:1c:d0")
	}
	if en0.State != "active" {
		t.Errorf("en0.State = %q, want %q", en0.State, "active")
	}
	if len(en0.IPv4Addresses) != 1 {
		t.Fatalf("en0 IPv4 count = %d, want 1", len(en0.IPv4Addresses))
	}
	if en0.IPv4Addresses[0].Address != "192.168.1.100" {
		t.Errorf("en0 IPv4 = %q, want %q", en0.IPv4Addresses[0].Address, "192.168.1.100")
	}
	if en0.IPv4Addresses[0].PrefixLen != 24 {
		t.Errorf("en0 IPv4 prefix = %d, want 24", en0.IPv4Addresses[0].PrefixLen)
	}
	if len(en0.IPv6Addresses) != 2 {
		t.Fatalf("en0 IPv6 count = %d, want 2", len(en0.IPv6Addresses))
	}

	// en2: active with /16 netmask (en1 filtered: inactive + no IPv4; utun0 filtered: virtual).
	en2 := nics[1]
	if en2.Name != "en2" {
		t.Errorf("nics[1].Name = %q, want %q", en2.Name, "en2")
	}
	if len(en2.IPv4Addresses) != 1 {
		t.Fatalf("en2 IPv4 count = %d, want 1", len(en2.IPv4Addresses))
	}
	if en2.IPv4Addresses[0].PrefixLen != 16 {
		t.Errorf("en2 IPv4 prefix = %d, want 16", en2.IPv4Addresses[0].PrefixLen)
	}
}

func TestParseNetworksetupPorts(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/networksetup_ports.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	portMap := parseNetworksetupPorts(data)

	expected := map[string]string{
		"en0": "Wi-Fi",
		"en1": "Thunderbolt Ethernet Slot 1",
		"en2": "Thunderbolt Bridge",
	}

	if len(portMap) != len(expected) {
		t.Fatalf("expected %d ports, got %d: %v", len(expected), len(portMap), portMap)
	}

	for dev, want := range expected {
		got, ok := portMap[dev]
		if !ok {
			t.Errorf("missing device %q in port map", dev)
			continue
		}
		if got != want {
			t.Errorf("portMap[%q] = %q, want %q", dev, got, want)
		}
	}
}

func TestParseIoregBattery(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/ioreg_battery.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	info := parseIoregBattery(data)

	if !info.Present {
		t.Fatal("expected battery to be present")
	}
	if info.CycleCount != 342 {
		t.Errorf("CycleCount = %d, want 342", info.CycleCount)
	}
	// CapacityPct = 3825 * 100 / 4521 (AppleRawMaxCapacity) = 84
	if info.CapacityPct != 84 {
		t.Errorf("CapacityPct = %d, want 84", info.CapacityPct)
	}
	// HealthPct = 4521 * 100 / 5000 = 90
	if info.HealthPct != 90 {
		t.Errorf("HealthPct = %d, want 90", info.HealthPct)
	}
	if info.Status != "Discharging" {
		t.Errorf("Status = %q, want %q", info.Status, "Discharging")
	}
	if info.Technology != "lithium-ion" {
		t.Errorf("Technology = %q, want %q", info.Technology, "lithium-ion")
	}
}

func TestParseIoregBattery_NoBattery(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/ioreg_battery_none.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	info := parseIoregBattery(data)

	if info.Present {
		t.Error("expected battery to not be present for empty input")
	}
}

func TestCollectDarwinVirtualization_Detection(t *testing.T) {
	tests := []struct {
		model  string
		isVirt bool
		hyperv string
	}{
		{"Mac14,10", false, ""},
		{"MacBookPro18,1", false, ""},
		{"VMware7,1", true, "vmware"},
		{"VirtualBox", true, "virtualbox"},
		{"Parallels15,1", true, "parallels"},
		{"QEMU Virtual Machine", true, "qemu"},
		{"iMac21,1", false, ""},
	}

	for _, tt := range tests {
		got := detectVirtFromModel(tt.model)
		if got.IsVirtual != tt.isVirt {
			t.Errorf("detectVirtFromModel(%q).IsVirtual = %v, want %v", tt.model, got.IsVirtual, tt.isVirt)
		}
		if got.HypervisorType != tt.hyperv {
			t.Errorf("detectVirtFromModel(%q).HypervisorType = %q, want %q", tt.model, got.HypervisorType, tt.hyperv)
		}
	}
}

func TestParseSPDisplaysDataType(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/sp_displays.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	var parsed spDisplaysData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	gpus := parseSPDisplaysDataType(parsed)

	if len(gpus) != 2 {
		t.Fatalf("expected 2 GPUs, got %d", len(gpus))
	}

	// First: Apple M2 Pro with shared VRAM.
	if gpus[0].Model != "Apple M2 Pro" {
		t.Errorf("gpus[0].Model = %q, want %q", gpus[0].Model, "Apple M2 Pro")
	}
	// 24576 MB
	if gpus[0].VRAMMB != 24576 {
		t.Errorf("gpus[0].VRAMMB = %d, want 24576", gpus[0].VRAMMB)
	}

	// Second: Radeon Pro 5500M with dedicated VRAM.
	if gpus[1].Model != "Radeon Pro 5500M" {
		t.Errorf("gpus[1].Model = %q, want %q", gpus[1].Model, "Radeon Pro 5500M")
	}
	if gpus[1].VRAMMB != 4096 {
		t.Errorf("gpus[1].VRAMMB = %d, want 4096", gpus[1].VRAMMB)
	}
	if gpus[1].PCISlot != "Slot - 0x02" {
		t.Errorf("gpus[1].PCISlot = %q, want %q", gpus[1].PCISlot, "Slot - 0x02")
	}
}

func TestParseSPStorageDataType(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/sp_storage.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	var parsed spStorageData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	devices := parseSPStorageDataType(context.Background(), parsed)

	// 4 volumes grouped into 3 physical drives:
	// "APPLE SSD AP1024Z" (2 volumes), "Samsung T7" (1 volume), "WD Black SN850" (1 volume).
	if len(devices) != 3 {
		t.Fatalf("expected 3 storage devices, got %d", len(devices))
	}

	// Apple SSD — should have 2 partitions.
	apple := devices[0]
	if apple.Model != "APPLE SSD AP1024Z" {
		t.Errorf("devices[0].Model = %q, want %q", apple.Model, "APPLE SSD AP1024Z")
	}
	if apple.Type != "nvme" {
		t.Errorf("devices[0].Type = %q, want %q", apple.Type, "nvme")
	}
	if len(apple.Partitions) != 2 {
		t.Fatalf("devices[0] partition count = %d, want 2", len(apple.Partitions))
	}
	if apple.Partitions[0].FSType != "APFS" {
		t.Errorf("partition[0].FSType = %q, want %q", apple.Partitions[0].FSType, "APFS")
	}
	if apple.Partitions[0].MountPoint != "/" {
		t.Errorf("partition[0].MountPoint = %q, want %q", apple.Partitions[0].MountPoint, "/")
	}

	// Samsung T7 — USB external.
	samsung := devices[1]
	if samsung.Model != "Samsung T7" {
		t.Errorf("devices[1].Model = %q, want %q", samsung.Model, "Samsung T7")
	}
	if samsung.Transport != "USB" {
		t.Errorf("devices[1].Transport = %q, want %q", samsung.Transport, "USB")
	}
	if len(samsung.Partitions) != 1 {
		t.Fatalf("devices[1] partition count = %d, want 1", len(samsung.Partitions))
	}

	// WD Black SN850 — NVMe.
	wd := devices[2]
	if wd.Type != "nvme" {
		t.Errorf("devices[2].Type = %q, want %q", wd.Type, "nvme")
	}
}

func TestParseSPUSBDataType(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/sp_usb.json")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	var parsed spUSBData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	devices := parseSPUSBDataType(parsed)

	// Bus 1: USB Keyboard, USB Hub, (nested) USB Mouse, Webcam
	// Bus 2: External SSD
	// Total: 5 devices (hub itself counts as a device too).
	if len(devices) != 5 {
		t.Fatalf("expected 5 USB devices, got %d", len(devices))
	}

	// Verify first device.
	if devices[0].Description != "USB Keyboard" {
		t.Errorf("devices[0].Description = %q, want %q", devices[0].Description, "USB Keyboard")
	}
	if devices[0].VendorID != "0x05ac" {
		t.Errorf("devices[0].VendorID = %q, want %q", devices[0].VendorID, "0x05ac")
	}
	if devices[0].ProductID != "0x0084" {
		t.Errorf("devices[0].ProductID = %q, want %q", devices[0].ProductID, "0x0084")
	}

	// USB Hub is device[1].
	if devices[1].Description != "USB Hub" {
		t.Errorf("devices[1].Description = %q, want %q", devices[1].Description, "USB Hub")
	}

	// Nested devices under hub.
	if devices[2].Description != "USB Mouse" {
		t.Errorf("devices[2].Description = %q, want %q", devices[2].Description, "USB Mouse")
	}
	if devices[3].Description != "Webcam" {
		t.Errorf("devices[3].Description = %q, want %q", devices[3].Description, "Webcam")
	}

	// Second bus.
	if devices[4].Description != "External SSD" {
		t.Errorf("devices[4].Description = %q, want %q", devices[4].Description, "External SSD")
	}
}

func TestParseVmStat_Hardware(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/vm_stat_apple_silicon.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pageSize, pages := parseVmStat(data)

	if pageSize != 16384 {
		t.Errorf("pageSize = %d, want 16384", pageSize)
	}
	if pages["pages free"] != 12345 {
		t.Errorf("pages free = %d, want 12345", pages["pages free"])
	}
	if pages["pages active"] != 56789 {
		t.Errorf("pages active = %d, want 56789", pages["pages active"])
	}
	if pages["pages inactive"] != 23456 {
		t.Errorf("pages inactive = %d, want 23456", pages["pages inactive"])
	}
	if pages["pages wired down"] != 34567 {
		t.Errorf("pages wired down = %d, want 34567", pages["pages wired down"])
	}
}
