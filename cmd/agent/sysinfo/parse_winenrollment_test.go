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
