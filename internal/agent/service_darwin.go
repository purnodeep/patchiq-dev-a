//go:build darwin

package agent

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/patchiq-agent.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/patchiq-agent.err</string>
</dict>
</plist>
`

// DarwinService manages the PatchIQ agent launchd daemon.
type DarwinService struct {
	Label      string
	BinaryPath string
}

// PlistPath returns the standard launchd plist path.
func (s *DarwinService) PlistPath() string {
	return "/Library/LaunchDaemons/" + s.Label + ".plist"
}

// GeneratePlist returns the plist XML for this service.
func (s *DarwinService) GeneratePlist() string {
	return fmt.Sprintf(plistTemplate, s.Label, s.BinaryPath)
}

// Install writes the plist file.
func (s *DarwinService) Install() error {
	plist := s.GeneratePlist()
	if err := os.WriteFile(s.PlistPath(), []byte(plist), 0644); err != nil {
		return fmt.Errorf("write plist %s: %w", s.PlistPath(), err)
	}
	return nil
}

// Start loads the daemon via launchctl.
func (s *DarwinService) Start() error {
	out, err := exec.Command("launchctl", "load", s.PlistPath()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %s: %w", string(out), err)
	}
	return nil
}

// Stop unloads the daemon via launchctl.
func (s *DarwinService) Stop() error {
	out, err := exec.Command("launchctl", "unload", s.PlistPath()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl unload: %s: %w", string(out), err)
	}
	return nil
}

// Uninstall stops the daemon and removes the plist.
func (s *DarwinService) Uninstall() error {
	_ = s.Stop()
	if err := os.Remove(s.PlistPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist %s: %w", s.PlistPath(), err)
	}
	return nil
}

// Status returns the daemon status from launchctl.
func (s *DarwinService) Status() (string, error) {
	out, err := exec.Command("launchctl", "list").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("launchctl list: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, s.Label) {
			return strings.TrimSpace(line), nil
		}
	}
	return "not running", nil
}
