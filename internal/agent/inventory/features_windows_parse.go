//go:build windows

package inventory

import (
	"encoding/json"
	"fmt"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
)

type optionalFeature struct {
	FeatureName string `json:"FeatureName"`
	State       int    `json:"State"`
}

type serverFeature struct {
	Name         string `json:"Name"`
	InstallState string `json:"InstallState"`
}

// parseWindowsOptionalFeatures parses Get-WindowsOptionalFeature JSON output.
// State values: 0=Disabled, 1=Enabled, 2=DisabledWithPayloadRemoved.
func parseWindowsOptionalFeatures(data []byte) ([]*pb.PackageInfo, error) {
	var features []optionalFeature
	// Handle single object vs array.
	if err := json.Unmarshal(data, &features); err != nil {
		var single optionalFeature
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("parse windows features: %w", err)
		}
		features = []optionalFeature{single}
	}

	var pkgs []*pb.PackageInfo
	for _, f := range features {
		if f.FeatureName == "" {
			continue
		}
		status := "Disabled"
		if f.State == 1 {
			status = "Enabled"
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:   f.FeatureName,
			Source: "windows_feature",
			Status: status,
		})
	}
	return pkgs, nil
}

// parseWindowsServerFeatures parses Get-WindowsFeature JSON output (Server SKU).
func parseWindowsServerFeatures(data []byte) ([]*pb.PackageInfo, error) {
	var features []serverFeature
	if err := json.Unmarshal(data, &features); err != nil {
		var single serverFeature
		if err2 := json.Unmarshal(data, &single); err2 != nil {
			return nil, fmt.Errorf("parse server features: %w", err)
		}
		features = []serverFeature{single}
	}

	var pkgs []*pb.PackageInfo
	for _, f := range features {
		if f.Name == "" {
			continue
		}
		pkgs = append(pkgs, &pb.PackageInfo{
			Name:   f.Name,
			Source: "windows_feature",
			Status: f.InstallState,
		})
	}
	return pkgs, nil
}
