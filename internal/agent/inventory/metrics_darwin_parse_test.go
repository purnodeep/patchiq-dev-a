//go:build darwin

package inventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readDarwinMetricsTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "darwin", name))
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	return data
}

// parseVmStat is already tested in hardware_darwin_parse_test.go:
// TestParseVmStat (Apple Silicon) and TestParseVmStat_Intel.
// Additional coverage for empty input.
func TestParseVmStat_EmptyInput(t *testing.T) {
	pageSize, pages := parseVmStat([]byte{})
	if pageSize != 0 {
		t.Errorf("page size = %d, want 0", pageSize)
	}
	if len(pages) != 0 {
		t.Errorf("pages len = %d, want 0", len(pages))
	}
}

func TestParseSysctlLoadAvg(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "sysctl_loadavg.txt")
	l1, l5, l15 := parseSysctlLoadAvg(data)

	if l1 != 1.23 {
		t.Errorf("load avg 1 = %f, want 1.23", l1)
	}
	if l5 != 0.89 {
		t.Errorf("load avg 5 = %f, want 0.89", l5)
	}
	if l15 != 0.67 {
		t.Errorf("load avg 15 = %f, want 0.67", l15)
	}
}

func TestParseSysctlLoadAvg_Empty(t *testing.T) {
	l1, l5, l15 := parseSysctlLoadAvg([]byte{})
	if l1 != 0 || l5 != 0 || l15 != 0 {
		t.Errorf("expected all zeros, got %f %f %f", l1, l5, l15)
	}
}

func TestParseSysctlBoottime(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "sysctl_boottime.txt")
	sec := parseSysctlBoottime(data)
	if sec != 1710000000 {
		t.Errorf("boottime = %d, want 1710000000", sec)
	}
}

func TestParseSysctlBoottime_Empty(t *testing.T) {
	sec := parseSysctlBoottime([]byte{})
	if sec != 0 {
		t.Errorf("boottime = %d, want 0", sec)
	}
}

func TestParseSysctlSwap(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTotal uint64
		wantUsed  uint64
	}{
		{
			name:      "megabytes",
			input:     "vm.swapusage: total = 2048.00M  used = 512.00M  free = 1536.00M  (encrypted)",
			wantTotal: 2048 * 1024 * 1024,
			wantUsed:  512 * 1024 * 1024,
		},
		{
			name:      "gigabytes",
			input:     "vm.swapusage: total = 4.00G  used = 1.50G  free = 2.50G",
			wantTotal: 4 * 1024 * 1024 * 1024,
			wantUsed:  uint64(1.5 * 1024 * 1024 * 1024),
		},
		{
			name:      "empty",
			input:     "",
			wantTotal: 0,
			wantUsed:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, used := parseSysctlSwap([]byte(tt.input))
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
			if used != tt.wantUsed {
				t.Errorf("used = %d, want %d", used, tt.wantUsed)
			}
		})
	}
}

func TestParseSysctlSwap_Testdata(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "sysctl_swapusage.txt")
	total, used := parseSysctlSwap(data)
	wantTotal := uint64(2048 * 1024 * 1024)
	wantUsed := uint64(512 * 1024 * 1024)
	if total != wantTotal {
		t.Errorf("total = %d, want %d", total, wantTotal)
	}
	if used != wantUsed {
		t.Errorf("used = %d, want %d", used, wantUsed)
	}
}

func TestParseDarwinTopCPU(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "top_cpu.txt")
	user, sys, idle := parseDarwinTopCPU(data)

	// Should return the SECOND CPU usage line values.
	if user != 12.50 {
		t.Errorf("user = %f, want 12.50", user)
	}
	if sys != 8.33 {
		t.Errorf("sys = %f, want 8.33", sys)
	}
	if idle != 79.16 {
		t.Errorf("idle = %f, want 79.16", idle)
	}
}

func TestParseDarwinTopCPU_SingleSample(t *testing.T) {
	input := []byte("CPU usage: 25.00% user, 10.00% sys, 65.00% idle\n")
	user, sys, idle := parseDarwinTopCPU(input)

	if user != 25.00 {
		t.Errorf("user = %f, want 25.00", user)
	}
	if sys != 10.00 {
		t.Errorf("sys = %f, want 10.00", sys)
	}
	if idle != 65.00 {
		t.Errorf("idle = %f, want 65.00", idle)
	}
}

func TestParseDarwinTopCPU_Empty(t *testing.T) {
	user, sys, idle := parseDarwinTopCPU([]byte{})
	if user != 0 || sys != 0 || idle != 0 {
		t.Errorf("expected all zeros, got user=%f sys=%f idle=%f", user, sys, idle)
	}
}

func TestParseNetstatIb(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "netstat_ib.txt")
	samples := parseNetstatIb(data)

	// Should have en0 and en1, but NOT lo0.
	if len(samples) != 2 {
		t.Fatalf("got %d samples, want 2", len(samples))
	}

	for _, s := range samples {
		if s.iface == "lo0" {
			t.Error("lo0 should be skipped")
		}
	}

	// Check en0.
	en0 := samples[0]
	if en0.iface != "en0" {
		t.Errorf("first interface = %q, want en0", en0.iface)
	}
	if en0.rxBytes != 67890123 {
		t.Errorf("en0 rxBytes = %d, want 67890123", en0.rxBytes)
	}
	if en0.txBytes != 98765432 {
		t.Errorf("en0 txBytes = %d, want 98765432", en0.txBytes)
	}
	if en0.rxPackets != 99999 {
		t.Errorf("en0 rxPackets = %d, want 99999", en0.rxPackets)
	}
	if en0.txPackets != 54321 {
		t.Errorf("en0 txPackets = %d, want 54321", en0.txPackets)
	}

	// Check en1.
	en1 := samples[1]
	if en1.iface != "en1" {
		t.Errorf("second interface = %q, want en1", en1.iface)
	}
	if en1.rxBytes != 2222222 {
		t.Errorf("en1 rxBytes = %d, want 2222222", en1.rxBytes)
	}
	if en1.txBytes != 4444444 {
		t.Errorf("en1 txBytes = %d, want 4444444", en1.txBytes)
	}
}

func TestParseNetstatIb_Empty(t *testing.T) {
	samples := parseNetstatIb([]byte{})
	if len(samples) != 0 {
		t.Errorf("got %d samples, want 0", len(samples))
	}
}

func TestCalcDarwinNetIO(t *testing.T) {
	s1 := []darwinNetSample{
		{iface: "en0", rxBytes: 1000, txBytes: 2000, rxPackets: 10, txPackets: 20},
		{iface: "en1", rxBytes: 500, txBytes: 600, rxPackets: 5, txPackets: 6},
	}
	s2 := []darwinNetSample{
		{iface: "en0", rxBytes: 1200, txBytes: 2400, rxPackets: 12, txPackets: 24},
		{iface: "en1", rxBytes: 700, txBytes: 800, rxPackets: 7, txPackets: 8},
	}

	metrics := calcDarwinNetIO(s1, s2)
	if len(metrics) != 2 {
		t.Fatalf("got %d metrics, want 2", len(metrics))
	}

	// en0: rxBytes delta=200/0.2=1000, txBytes delta=400/0.2=2000.
	en0 := metrics[0]
	if en0.Interface != "en0" {
		t.Errorf("first interface = %q, want en0", en0.Interface)
	}
	if en0.RxBytesPS != 1000 {
		t.Errorf("en0 RxBytesPS = %f, want 1000", en0.RxBytesPS)
	}
	if en0.TxBytesPS != 2000 {
		t.Errorf("en0 TxBytesPS = %f, want 2000", en0.TxBytesPS)
	}
	if en0.RxPacketsPS != 10 {
		t.Errorf("en0 RxPacketsPS = %f, want 10", en0.RxPacketsPS)
	}
	if en0.TxPacketsPS != 20 {
		t.Errorf("en0 TxPacketsPS = %f, want 20", en0.TxPacketsPS)
	}

	// en1: rxBytes delta=200/0.2=1000, txBytes delta=200/0.2=1000.
	en1 := metrics[1]
	if en1.Interface != "en1" {
		t.Errorf("second interface = %q, want en1", en1.Interface)
	}
	if en1.RxBytesPS != 1000 {
		t.Errorf("en1 RxBytesPS = %f, want 1000", en1.RxBytesPS)
	}
	if en1.TxBytesPS != 1000 {
		t.Errorf("en1 TxBytesPS = %f, want 1000", en1.TxBytesPS)
	}
}

func TestCalcDarwinNetIO_Empty(t *testing.T) {
	metrics := calcDarwinNetIO(nil, nil)
	if len(metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(metrics))
	}
}

func TestCalcDarwinNetIO_MismatchedInterfaces(t *testing.T) {
	s1 := []darwinNetSample{
		{iface: "en0", rxBytes: 1000, txBytes: 2000, rxPackets: 10, txPackets: 20},
	}
	s2 := []darwinNetSample{
		{iface: "en1", rxBytes: 1200, txBytes: 2400, rxPackets: 12, txPackets: 24},
	}
	metrics := calcDarwinNetIO(s1, s2)
	if len(metrics) != 0 {
		t.Errorf("got %d metrics for mismatched interfaces, want 0", len(metrics))
	}
}

func TestParseMountOutput(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "mount.txt")
	types := parseMountOutput(data)

	if types["/"] != "apfs" {
		t.Errorf("root fs type = %q, want apfs", types["/"])
	}
	if types["/dev"] != "devfs" {
		t.Errorf("/dev fs type = %q, want devfs", types["/dev"])
	}
	if types["/System/Volumes/Data/home"] != "autofs" {
		t.Errorf("/System/Volumes/Data/home fs type = %q, want autofs", types["/System/Volumes/Data/home"])
	}
	if types["/System/Volumes/Data"] != "apfs" {
		t.Errorf("/System/Volumes/Data fs type = %q, want apfs", types["/System/Volumes/Data"])
	}
}

func TestParseMountOutput_Empty(t *testing.T) {
	types := parseMountOutput([]byte{})
	if len(types) != 0 {
		t.Errorf("got %d entries, want 0", len(types))
	}
}

func TestParseDfPk(t *testing.T) {
	dfData := readDarwinMetricsTestdata(t, "df_pk.txt")
	mountData := readDarwinMetricsTestdata(t, "mount.txt")
	mountTypes := parseMountOutput(mountData)

	metrics := parseDfPk(dfData, mountTypes)

	// Should include real /dev/* mounts but NOT devfs or map auto_home.
	for _, m := range metrics {
		if m.Device == "devfs" || m.Device == "map" {
			t.Errorf("pseudo-filesystem %q should be filtered out", m.Device)
		}
		if !strings.HasPrefix(m.Device, "/") {
			t.Errorf("non-block device %q should be filtered out", m.Device)
		}
	}

	// Expect: /, /System/Volumes/VM, /System/Volumes/Preboot, /System/Volumes/Data, /System/Volumes/xarts
	if len(metrics) != 5 {
		t.Fatalf("got %d filesystems, want 5; metrics: %+v", len(metrics), metrics)
	}

	// Check root filesystem.
	root := metrics[0]
	if root.Mount != "/" {
		t.Errorf("first mount = %q, want /", root.Mount)
	}
	if root.Device != "/dev/disk3s1s1" {
		t.Errorf("root device = %q, want /dev/disk3s1s1", root.Device)
	}
	if root.FSType != "apfs" {
		t.Errorf("root fs type = %q, want apfs", root.FSType)
	}
	// 12164028 * 1024 = 12455964672
	if root.TotalBytes != 482797652*1024 {
		t.Errorf("root total bytes = %d, want %d", root.TotalBytes, 482797652*1024)
	}
	if root.UsedBytes != 12164028*1024 {
		t.Errorf("root used bytes = %d, want %d", root.UsedBytes, 12164028*1024)
	}
	if root.AvailBytes != 214621688*1024 {
		t.Errorf("root avail bytes = %d, want %d", root.AvailBytes, 214621688*1024)
	}
	if root.UsePct != 6 {
		t.Errorf("root use pct = %f, want 6", root.UsePct)
	}
}

func TestParseDfPk_Empty(t *testing.T) {
	metrics := parseDfPk([]byte{}, nil)
	if len(metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(metrics))
	}
}

func TestParseDfPk_NoMountTypes(t *testing.T) {
	dfData := readDarwinMetricsTestdata(t, "df_pk.txt")
	metrics := parseDfPk(dfData, nil)

	// Should still work, just without fs type.
	if len(metrics) != 5 {
		t.Fatalf("got %d filesystems, want 5", len(metrics))
	}
	if metrics[0].FSType != "" {
		t.Errorf("expected empty fs type without mount data, got %q", metrics[0].FSType)
	}
}

func TestParseIostat(t *testing.T) {
	data := readDarwinMetricsTestdata(t, "iostat_d.txt")
	metrics := parseIostat(data)

	if len(metrics) != 2 {
		t.Fatalf("got %d metrics, want 2", len(metrics))
	}

	// Second data row: disk0 = 0.08 MB/s, disk1 = 0.02 MB/s.
	disk0 := metrics[0]
	if disk0.Device != "disk0" {
		t.Errorf("first device = %q, want disk0", disk0.Device)
	}
	wantRead0 := 0.08 * 1048576
	if disk0.ReadBytesPS != wantRead0 {
		t.Errorf("disk0 ReadBytesPS = %f, want %f", disk0.ReadBytesPS, wantRead0)
	}
	if disk0.WriteBytesPS != 0 {
		t.Errorf("disk0 WriteBytesPS = %f, want 0", disk0.WriteBytesPS)
	}

	disk1 := metrics[1]
	if disk1.Device != "disk1" {
		t.Errorf("second device = %q, want disk1", disk1.Device)
	}
	wantRead1 := 0.02 * 1048576
	if disk1.ReadBytesPS != wantRead1 {
		t.Errorf("disk1 ReadBytesPS = %f, want %f", disk1.ReadBytesPS, wantRead1)
	}
}

func TestParseIostat_Empty(t *testing.T) {
	metrics := parseIostat([]byte{})
	if len(metrics) != 0 {
		t.Errorf("got %d metrics, want 0", len(metrics))
	}
}
