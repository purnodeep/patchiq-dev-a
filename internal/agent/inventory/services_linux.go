//go:build linux

package inventory

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// CollectServices collects systemd service unit information on Linux.
// It runs `systemctl list-units --type=service --all --no-pager --plain`
// and checks each loaded service's enabled state.
func CollectServices(ctx context.Context, logger *slog.Logger) ([]ServiceInfo, error) {
	runner := &execRunner{}

	out, err := runner.Run(ctx, "systemctl", "list-units", "--type=service", "--all", "--no-pager", "--plain")
	if err != nil {
		return nil, fmt.Errorf("collect services: list-units: %w", err)
	}

	services := parseSystemctlListUnits(string(out))

	// Categorize and check enabled state for each service.
	for i := range services {
		services[i].Category = categorizeService(services[i].Name)
	}
	for i := range services {
		enabled, checkErr := checkServiceEnabled(ctx, runner, services[i].Name)
		if checkErr != nil {
			if logger != nil {
				logger.Debug("failed to check service enabled state",
					"service", services[i].Name,
					"error", checkErr,
				)
			}
			continue
		}
		services[i].Enabled = enabled
	}

	return services, nil
}

// parseSystemctlListUnits parses the output of
// `systemctl list-units --type=service --all --no-pager --plain`.
//
// Example output:
//
//	UNIT                     LOAD      ACTIVE   SUB     DESCRIPTION
//	accounts-daemon.service  loaded    active   running Accounts Service
//	alsa-restore.service     loaded    inactive dead    Save/Restore Sound Card State
//	apparmor.service         not-found inactive dead    apparmor.service
func parseSystemctlListUnits(data string) []ServiceInfo {
	var services []ServiceInfo

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip header line.
		if strings.HasPrefix(line, "UNIT") {
			continue
		}

		// Stop at the summary footer (e.g., "123 loaded units listed.").
		if strings.Contains(line, " loaded units listed") || strings.Contains(line, "To show all installed") {
			break
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		unit := fields[0]
		loadState := fields[1]
		activeState := fields[2]
		subState := fields[3]

		// Only include loaded services.
		if loadState != "loaded" {
			continue
		}

		// Extract service name (remove .service suffix).
		name := strings.TrimSuffix(unit, ".service")

		// Description is everything after the first 4 fields.
		description := ""
		if len(fields) > 4 {
			description = strings.Join(fields[4:], " ")
		}

		services = append(services, ServiceInfo{
			Name:        name,
			Description: description,
			LoadState:   loadState,
			ActiveState: activeState,
			SubState:    subState,
		})
	}

	return services
}

// categorizeService returns a category for a systemd service based on name patterns.
func categorizeService(name string) string {
	prefixes := []struct {
		patterns []string
		category string
	}{
		{[]string{"systemd-", "dbus", "udev"}, "System"},
		{[]string{"docker", "containerd"}, "Container"},
		{[]string{"postgres", "mysql", "redis", "valkey", "mongo"}, "Database"},
		{[]string{"NetworkManager", "ssh", "firewall", "nginx", "apache", "wpa"}, "Network"},
		{[]string{"bluetooth", "cups", "alsa", "nvidia", "gpu"}, "Hardware"},
		{[]string{"grafana", "prometheus", "otel", "node_exporter"}, "Monitoring"},
		{[]string{"actions.runner.", "air-", "jenkins", "gitlab"}, "CI/CD"},
		{[]string{"snap", "packagekit"}, "Package Management"},
		{[]string{"apparmor", "ufw", "fail2ban", "polkit"}, "Security"},
		{[]string{"cron", "timer", "logrotate", "rsyslog"}, "Maintenance"},
	}
	for _, group := range prefixes {
		for _, p := range group.patterns {
			if strings.HasPrefix(name, p) {
				return group.category
			}
		}
	}
	return "Other"
}

// checkServiceEnabled runs `systemctl is-enabled <name>` and returns true if
// the service is enabled. systemctl is-enabled returns non-zero for
// disabled/masked/static services but always writes the state string to stdout.
// We check the output regardless of exit code, and only return an error on
// genuine execution failures (e.g. systemctl not found, permission denied).
func checkServiceEnabled(ctx context.Context, runner commandRunner, name string) (bool, error) {
	out, err := runner.Run(ctx, "systemctl", "is-enabled", name+".service")
	// systemctl is-enabled returns non-zero for disabled/masked/static services
	// but always writes the state string to stdout. Check output regardless of exit code.
	state := strings.TrimSpace(string(out))
	if state == "enabled" || state == "enabled-runtime" {
		return true, nil
	}
	if err != nil {
		// Genuine execution failure (systemctl not found, permission denied, etc.)
		return false, err
	}
	return false, nil
}
