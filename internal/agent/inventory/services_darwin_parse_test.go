package inventory

import (
	"os"
	"testing"
)

func TestParseLaunchctlList(t *testing.T) {
	data, err := os.ReadFile("testdata/darwin/launchctl_list.txt")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	services := parseLaunchctlList(string(data))

	if len(services) != 12 {
		t.Fatalf("expected 12 services, got %d", len(services))
	}

	tests := []struct {
		idx         int
		name        string
		loadState   string
		activeState string
		subState    string
		enabled     bool
		description string
	}{
		{0, "com.apple.Spotlight", "loaded", "active", "running", true, ""},
		{1, "com.apple.Accessibility.AXUIAutomation", "loaded", "inactive", "dead", true, ""},
		{2, "org.homebrew.mxcl.postgresql@14", "loaded", "active", "running", true, ""},
		{3, "com.apple.xprotect.check", "loaded", "inactive", "dead", true, ""},
		{4, "com.docker.vmnetd", "loaded", "active", "running", true, ""},
		{5, "com.microsoft.OneDrive", "loaded", "inactive", "dead", true, ""},
		{6, "com.openssh.sshd", "loaded", "active", "running", true, ""},
		{7, "org.postgres.pgctl", "loaded", "inactive", "dead", true, ""},
		{8, "com.apple.cron", "loaded", "active", "running", true, ""},
		{9, "com.apple.security.agent", "loaded", "inactive", "dead", true, ""},
		{10, "com.brave.Browser.updater", "loaded", "active", "running", true, ""},
		{11, "io.containerd.grpc.v1", "loaded", "inactive", "dead", true, ""},
	}

	for _, tt := range tests {
		s := services[tt.idx]
		if s.Name != tt.name {
			t.Errorf("services[%d].Name = %q, want %q", tt.idx, s.Name, tt.name)
		}
		if s.LoadState != tt.loadState {
			t.Errorf("services[%d].LoadState = %q, want %q", tt.idx, s.LoadState, tt.loadState)
		}
		if s.ActiveState != tt.activeState {
			t.Errorf("services[%d].ActiveState = %q, want %q", tt.idx, s.ActiveState, tt.activeState)
		}
		if s.SubState != tt.subState {
			t.Errorf("services[%d].SubState = %q, want %q", tt.idx, s.SubState, tt.subState)
		}
		if s.Enabled != tt.enabled {
			t.Errorf("services[%d].Enabled = %v, want %v", tt.idx, s.Enabled, tt.enabled)
		}
		if s.Description != tt.description {
			t.Errorf("services[%d].Description = %q, want %q", tt.idx, s.Description, tt.description)
		}
	}
}

func TestParseLaunchctlList_Empty(t *testing.T) {
	services := parseLaunchctlList("PID\tStatus\tLabel\n")

	if len(services) != 0 {
		t.Errorf("expected 0 services for header-only input, got %d", len(services))
	}
}

func TestParseLaunchctlList_MalformedLines(t *testing.T) {
	input := "PID\tStatus\tLabel\n" +
		"432\t0\tcom.apple.Spotlight\n" +
		"this line has no tabs\n" +
		"only\tone\ttab but ok\n" +
		"\t\t\n" + // three fields but empty label
		"-\t0\tcom.apple.valid\n"

	services := parseLaunchctlList(input)

	// Should parse: com.apple.Spotlight, "tab but ok" (3 fields, label is "tab but ok"), com.apple.valid
	// The "this line has no tabs" has no tabs so SplitN gives 1 field -> skipped.
	// The "\t\t\n" has 3 fields but empty label -> skipped.
	// "only\tone\ttab but ok" -> fields: ["only", "one", "tab but ok"] -> label="tab but ok", pid="only" which is not "-" and Atoi fails -> inactive/dead.
	expected := []struct {
		name        string
		activeState string
	}{
		{"com.apple.Spotlight", "active"},
		{"tab but ok", "inactive"},
		{"com.apple.valid", "inactive"},
	}

	if len(services) != len(expected) {
		t.Fatalf("expected %d services, got %d", len(expected), len(services))
	}

	for i, tt := range expected {
		if services[i].Name != tt.name {
			t.Errorf("services[%d].Name = %q, want %q", i, services[i].Name, tt.name)
		}
		if services[i].ActiveState != tt.activeState {
			t.Errorf("services[%d].ActiveState = %q, want %q", i, services[i].ActiveState, tt.activeState)
		}
	}
}

func TestCategorizeDarwinService(t *testing.T) {
	tests := []struct {
		label    string
		category string
	}{
		// Security (specific Apple prefixes before general com.apple.)
		{"com.apple.security.agent", "Security"},
		{"com.apple.security.pboxd", "Security"},
		{"com.apple.xprotect.check", "Security"},
		{"com.apple.xprotect.daemon", "Security"},
		{"com.apple.ManagedClient.agent", "Security"},

		// System (general com.apple.)
		{"com.apple.Spotlight", "System"},
		{"com.apple.Finder", "System"},
		{"com.apple.Dock", "System"},

		// Package Management
		{"org.homebrew.mxcl.postgresql@14", "Package Management"},
		{"org.homebrew.mxcl.redis", "Package Management"},

		// Container
		{"com.docker.vmnetd", "Container"},
		{"com.docker.helper", "Container"},
		{"io.containerd.grpc.v1", "Container"},

		// Application
		{"com.microsoft.OneDrive", "Application"},
		{"com.google.Chrome", "Application"},
		{"org.mozilla.firefox", "Application"},
		{"com.brave.Browser.updater", "Application"},

		// Network
		{"com.openssh.sshd", "Network"},
		{"org.ntp.ntpd", "Network"},
		{"com.apple.NetworkDiagnostics", "Network"},

		// Database
		{"org.postgres.pgctl", "Database"},
		{"org.postgresql.pgctl", "Database"},
		{"com.redis.server", "Database"},
		{"com.mongo.mongod", "Database"},
		{"io.valkey.server", "Database"},

		// Maintenance
		{"com.apple.cron", "Maintenance"},
		{"com.apple.periodic-daily", "Maintenance"},

		// Other (no match)
		{"com.example.custom", "Other"},
		{"my.custom.service", "Other"},
	}

	for _, tt := range tests {
		got := categorizeDarwinService(tt.label)
		if got != tt.category {
			t.Errorf("categorizeDarwinService(%q) = %q, want %q", tt.label, got, tt.category)
		}
	}
}
