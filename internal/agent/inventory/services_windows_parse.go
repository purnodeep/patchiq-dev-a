package inventory

import (
	"encoding/json"
	"strings"
)

type winServiceEntry struct {
	Name        string `json:"Name"`
	DisplayName string `json:"DisplayName"`
	Status      int    `json:"Status"`
	StartType   int    `json:"StartType"`
}

// parseWinServices parses Get-Service JSON output into ServiceInfo slices.
// Status: 1=Stopped, 4=Running, 7=Paused.
// StartType: 0=Boot, 1=System, 2=Automatic, 3=Manual, 4=Disabled.
func parseWinServices(data string) []ServiceInfo {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil
	}

	var entries []winServiceEntry
	if strings.HasPrefix(data, "[") {
		if err := json.Unmarshal([]byte(data), &entries); err != nil {
			return nil
		}
	} else {
		var single winServiceEntry
		if err := json.Unmarshal([]byte(data), &single); err != nil {
			return nil
		}
		entries = []winServiceEntry{single}
	}

	services := make([]ServiceInfo, 0, len(entries))
	for _, e := range entries {
		activeState, subState := winServiceState(e.Status)
		svc := ServiceInfo{
			Name:        e.Name,
			Description: e.DisplayName,
			LoadState:   "loaded",
			ActiveState: activeState,
			SubState:    subState,
			Enabled:     e.StartType == 0 || e.StartType == 1 || e.StartType == 2,
			Category:    categorizeWinService(e.Name),
		}
		services = append(services, svc)
	}

	return services
}

func winServiceState(status int) (activeState, subState string) {
	switch status {
	case 4:
		return "active", "running"
	case 1:
		return "inactive", "dead"
	case 7:
		return "inactive", "paused"
	default:
		return "activating", "start-pending"
	}
}

func categorizeWinService(name string) string {
	lower := strings.ToLower(name)
	patterns := []struct {
		matches  []string
		category string
	}{
		{[]string{"windefend", "securityhealth", "mpssvc", "wscsvc"}, "Security"},
		{[]string{"mssql", "mysql", "postgres", "mongodb", "redis", "valkey"}, "Database"},
		{[]string{"w32time", "dnscache", "winrm", "sshd", "dhcp", "netbt"}, "Network"},
		{[]string{"eventlog", "winmgmt", "diagtrack"}, "Monitoring"},
		{[]string{"docker", "containerd"}, "Container"},
		{[]string{"bits", "wuauserv", "trustedinstaller", "msiserver"}, "Package Management"},
		{[]string{"spooler", "audioendpointbuilder", "audiosrv"}, "Hardware"},
		{[]string{"schedule"}, "Maintenance"},
	}

	for _, group := range patterns {
		for _, p := range group.matches {
			if strings.Contains(lower, p) {
				return group.category
			}
		}
	}
	return "Other"
}
