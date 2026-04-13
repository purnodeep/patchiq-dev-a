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
