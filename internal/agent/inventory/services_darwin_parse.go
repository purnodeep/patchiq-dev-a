package inventory

import (
	"strconv"
	"strings"
)

// parseLaunchctlList parses the output of `launchctl list`.
//
// Example output:
//
//	PID	Status	Label
//	432	0	com.apple.Spotlight
//	-	0	com.apple.Accessibility.AXUIAutomation
//	123	0	org.homebrew.mxcl.postgresql@14
//	-	78	com.apple.xprotect.check
func parseLaunchctlList(data string) []ServiceInfo {
	var services []ServiceInfo

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip the header line.
		if strings.HasPrefix(line, "PID") {
			continue
		}

		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}

		pidStr := strings.TrimSpace(fields[0])
		label := strings.TrimSpace(fields[2])

		if label == "" {
			continue
		}

		activeState := "inactive"
		subState := "dead"

		if pidStr != "-" {
			pid, err := strconv.Atoi(pidStr)
			if err == nil && pid > 0 {
				activeState = "active"
				subState = "running"
			}
		}

		services = append(services, ServiceInfo{
			Name:        label,
			Description: "",
			LoadState:   "loaded",
			ActiveState: activeState,
			SubState:    subState,
			Enabled:     true,
		})
	}

	return services
}

// categorizeDarwinService classifies a launchd service by its label prefix.
// More specific prefixes are checked before general ones (e.g., com.apple.security
// before com.apple.).
func categorizeDarwinService(label string) string {
	// Security-specific Apple services (must be checked before general com.apple.).
	securityPrefixes := []string{
		"com.apple.security",
		"com.apple.xprotect",
		"com.apple.ManagedClient",
	}
	for _, p := range securityPrefixes {
		if strings.HasPrefix(label, p) {
			return "Security"
		}
	}

	prefixes := []struct {
		patterns []string
		category string
	}{
		{[]string{"com.apple.NetworkDiagnostics"}, "Network"},
		{[]string{"com.apple.cron", "com.apple.periodic"}, "Maintenance"},
		{[]string{"com.apple."}, "System"},
		{[]string{"org.homebrew."}, "Package Management"},
		{[]string{"com.docker.", "io.containerd."}, "Container"},
		{[]string{"com.microsoft.", "com.google.", "org.mozilla.", "com.brave."}, "Application"},
		{[]string{"com.openssh.", "org.ntp."}, "Network"},
		{[]string{"org.postgres", "com.redis.", "com.mongo.", "io.valkey."}, "Database"},
	}

	for _, group := range prefixes {
		for _, p := range group.patterns {
			if strings.HasPrefix(label, p) {
				return group.category
			}
		}
	}

	return "Other"
}
