package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

type hotFixEntry struct {
	HotFixID    string          `json:"HotFixID"`
	Description string          `json:"Description"`
	InstalledOn json.RawMessage `json:"InstalledOn"`
}

// installedOnString extracts a date string from the InstalledOn field,
// which PowerShell may encode as a plain string, a /Date(...)/ string,
// or a datetime object {"value": "/Date(...)/", "DateTime": "..."}.
func installedOnString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try plain string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}

	// Try object with DateTime field
	var obj struct {
		DateTime string `json:"DateTime"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if obj.DateTime != "" {
			return obj.DateTime
		}
		return obj.Value
	}

	return string(raw)
}

// parseHotFixOutput parses JSON output from Get-HotFix | ConvertTo-Json.
// PowerShell emits a JSON array for multiple results or a single object for one result.
// Entries with empty HotFixID are skipped.
func parseHotFixOutput(data []byte) ([]*pb.PackageInfo, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}

	var entries []hotFixEntry

	if data[0] == '[' {
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("parse hotfix array: %w", err)
		}
	} else {
		var single hotFixEntry
		if err := json.Unmarshal(data, &single); err != nil {
			return nil, fmt.Errorf("parse hotfix object: %w", err)
		}
		entries = []hotFixEntry{single}
	}

	var pkgs []*pb.PackageInfo
	for _, e := range entries {
		if e.HotFixID == "" {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:        e.HotFixID,
			Version:     installedOnString(e.InstalledOn),
			Source:      "hotfix",
			Status:      e.Description,
			KbArticle:   e.HotFixID,
			InstallDate: installedOnString(e.InstalledOn),
		})
	}
	return pkgs, nil
}
