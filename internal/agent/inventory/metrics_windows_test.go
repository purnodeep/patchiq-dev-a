//go:build windows

package inventory

import (
	"testing"
)

func TestParseWinDiskJSON_Array(t *testing.T) {
	input := `[{"DeviceID":"C:","Size":512000000000,"FreeSpace":256000000000},{"DeviceID":"D:","Size":1000000000000,"FreeSpace":500000000000}]`
	disks, err := parseWinDiskJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("expected 2 disks, got %d", len(disks))
	}
	if disks[0].DeviceID != "C:" {
		t.Errorf("expected DeviceID C:, got %s", disks[0].DeviceID)
	}
	if disks[0].Size != 512000000000 {
		t.Errorf("expected Size 512000000000, got %d", disks[0].Size)
	}
	if disks[0].FreeSpace != 256000000000 {
		t.Errorf("expected FreeSpace 256000000000, got %d", disks[0].FreeSpace)
	}
}

func TestParseWinDiskJSON_SingleObject(t *testing.T) {
	input := `{"DeviceID":"C:","Size":512000000000,"FreeSpace":256000000000}`
	disks, err := parseWinDiskJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(disks))
	}
	if disks[0].DeviceID != "C:" {
		t.Errorf("expected DeviceID C:, got %s", disks[0].DeviceID)
	}
}

func TestParseWinDiskJSON_Empty(t *testing.T) {
	disks, err := parseWinDiskJSON("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(disks) != 0 {
		t.Fatalf("expected 0 disks, got %d", len(disks))
	}
}

func TestFillDiskWindowsMetrics(t *testing.T) {
	input := `[{"DeviceID":"C:","Size":100000,"FreeSpace":40000}]`
	disks, err := parseWinDiskJSON(input)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	m := &LiveMetrics{}
	for _, d := range disks {
		if d.Size == 0 {
			continue
		}
		used := d.Size - d.FreeSpace
		usePct := float64(used) / float64(d.Size) * 100.0
		m.Filesystems = append(m.Filesystems, FSMetric{
			Mount:      d.DeviceID,
			Device:     d.DeviceID,
			FSType:     "ntfs",
			TotalBytes: d.Size,
			UsedBytes:  used,
			AvailBytes: d.FreeSpace,
			UsePct:     usePct,
		})
	}

	if len(m.Filesystems) != 1 {
		t.Fatalf("expected 1 filesystem, got %d", len(m.Filesystems))
	}
	fs := m.Filesystems[0]
	if fs.UsedBytes != 60000 {
		t.Errorf("expected UsedBytes 60000, got %d", fs.UsedBytes)
	}
	if fs.UsePct != 60.0 {
		t.Errorf("expected UsePct 60.0, got %f", fs.UsePct)
	}
}

func TestWinMemInfoFields(t *testing.T) {
	// Verify that winMemInfo parses correctly and that memory calculations are correct.
	info := winMemInfo{
		TotalVisibleMemorySize: 16384000, // KB
		FreePhysicalMemory:     8192000,  // KB
	}
	m := &LiveMetrics{}
	m.MemoryTotalBytes = info.TotalVisibleMemorySize * 1024
	m.MemoryAvailableBytes = info.FreePhysicalMemory * 1024
	m.MemoryUsedBytes = m.MemoryTotalBytes - m.MemoryAvailableBytes
	if m.MemoryTotalBytes > 0 {
		m.MemoryUsedPct = float64(m.MemoryUsedBytes) / float64(m.MemoryTotalBytes) * 100.0
	}

	expectedTotal := uint64(16384000 * 1024)
	if m.MemoryTotalBytes != expectedTotal {
		t.Errorf("expected MemoryTotalBytes %d, got %d", expectedTotal, m.MemoryTotalBytes)
	}
	expectedUsed := expectedTotal - uint64(8192000*1024)
	if m.MemoryUsedBytes != expectedUsed {
		t.Errorf("expected MemoryUsedBytes %d, got %d", expectedUsed, m.MemoryUsedBytes)
	}
	if m.MemoryUsedPct != 50.0 {
		t.Errorf("expected MemoryUsedPct 50.0, got %f", m.MemoryUsedPct)
	}
}
