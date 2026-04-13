//go:build darwin

package agent

import (
	"strings"
	"testing"
)

func TestDarwinService_GeneratePlist(t *testing.T) {
	svc := &DarwinService{
		Label:      "com.patchiq.agent",
		BinaryPath: "/usr/local/bin/patchiq-agent",
	}

	plist := svc.GeneratePlist()

	checks := []string{
		"<key>Label</key>",
		"<string>com.patchiq.agent</string>",
		"<string>/usr/local/bin/patchiq-agent</string>",
		"<key>RunAtLoad</key>",
		"<key>KeepAlive</key>",
		"<key>StandardOutPath</key>",
		"<key>StandardErrorPath</key>",
	}

	for _, check := range checks {
		if !strings.Contains(plist, check) {
			t.Errorf("plist missing %q", check)
		}
	}
}

func TestDarwinService_PlistPath(t *testing.T) {
	svc := &DarwinService{Label: "com.patchiq.agent"}
	want := "/Library/LaunchDaemons/com.patchiq.agent.plist"
	if got := svc.PlistPath(); got != want {
		t.Errorf("PlistPath() = %q, want %q", got, want)
	}
}
