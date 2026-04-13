package compliance

import (
	"fmt"
	"testing"
)

func TestGetFramework(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantNil  bool
		wantName string
	}{
		{"NIST exists", FrameworkNIST80053, false, "NIST 800-53 Rev. 5"},
		{"PCI exists", FrameworkPCIDSSv4, false, "PCI DSS v4.0"},
		{"unknown returns nil", "unknown_framework", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := GetFramework(tt.id)
			if tt.wantNil {
				if fw != nil {
					t.Errorf("GetFramework(%q) = %v, want nil", tt.id, fw)
				}
				return
			}
			if fw == nil {
				t.Fatalf("GetFramework(%q) = nil, want framework", tt.id)
				return
			}
			if fw.Name != tt.wantName {
				t.Errorf("GetFramework(%q).Name = %q, want %q", tt.id, fw.Name, tt.wantName)
			}
		})
	}
}

func TestListFrameworks(t *testing.T) {
	frameworks := ListFrameworks()
	if len(frameworks) != 6 {
		t.Errorf("ListFrameworks() returned %d frameworks, want 6", len(frameworks))
	}
}

func TestFrameworkSLATimelines(t *testing.T) {
	tests := []struct {
		name        string
		frameworkID string
		controlID   string
		severity    string
		wantDays    *int
	}{
		{"NIST critical", FrameworkNIST80053, "SI-2", "critical", intPtr(15)},
		{"NIST high", FrameworkNIST80053, "SI-2", "high", intPtr(30)},
		{"NIST moderate", FrameworkNIST80053, "SI-2", "moderate", intPtr(90)},
		{"NIST low", FrameworkNIST80053, "SI-2", "low", nil},
		{"PCI critical", FrameworkPCIDSSv4, "6.3.3", "critical", intPtr(30)},
		{"PCI high", FrameworkPCIDSSv4, "6.3.3", "high", intPtr(30)},
		{"PCI moderate", FrameworkPCIDSSv4, "6.3.3", "moderate", nil},
		{"PCI low", FrameworkPCIDSSv4, "6.3.3", "low", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := GetFramework(tt.frameworkID)
			if fw == nil {
				t.Fatalf("framework %q not found", tt.frameworkID)
			}
			ctrl := fw.GetControl(tt.controlID)
			if ctrl == nil {
				t.Fatalf("control %q not found in framework %q", tt.controlID, tt.frameworkID)
			}
			got := ctrl.SLADays(tt.severity)
			if !intPtrEqual(got, tt.wantDays) {
				t.Errorf("SLADays(%q) = %v, want %v", tt.severity, intPtrStr(got), intPtrStr(tt.wantDays))
			}
		})
	}
}

func TestSLADaysByCVSS(t *testing.T) {
	tests := []struct {
		name         string
		frameworkID  string
		controlID    string
		cvss         float64
		wantDays     *int
		wantSeverity string
	}{
		{"NIST CVSS 10.0 critical", FrameworkNIST80053, "SI-2", 10.0, intPtr(15), "critical"},
		{"NIST CVSS 9.0 critical", FrameworkNIST80053, "SI-2", 9.0, intPtr(15), "critical"},
		{"NIST CVSS 8.9 high", FrameworkNIST80053, "SI-2", 8.9, intPtr(30), "high"},
		{"NIST CVSS 7.0 high", FrameworkNIST80053, "SI-2", 7.0, intPtr(30), "high"},
		{"NIST CVSS 6.9 moderate", FrameworkNIST80053, "SI-2", 6.9, intPtr(90), "moderate"},
		{"NIST CVSS 4.0 moderate", FrameworkNIST80053, "SI-2", 4.0, intPtr(90), "moderate"},
		{"NIST CVSS 3.9 low", FrameworkNIST80053, "SI-2", 3.9, nil, "low"},
		{"NIST CVSS 0.1 low", FrameworkNIST80053, "SI-2", 0.1, nil, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := GetFramework(tt.frameworkID)
			if fw == nil {
				t.Fatalf("framework %q not found", tt.frameworkID)
			}
			ctrl := fw.GetControl(tt.controlID)
			if ctrl == nil {
				t.Fatalf("control %q not found", tt.controlID)
			}
			gotDays, gotSeverity := ctrl.SLADaysByCVSS(tt.cvss)
			if !intPtrEqual(gotDays, tt.wantDays) {
				t.Errorf("SLADaysByCVSS(%v) days = %v, want %v", tt.cvss, intPtrStr(gotDays), intPtrStr(tt.wantDays))
			}
			if gotSeverity != tt.wantSeverity {
				t.Errorf("SLADaysByCVSS(%v) severity = %q, want %q", tt.cvss, gotSeverity, tt.wantSeverity)
			}
		})
	}
}

func TestPatchSLAControl(t *testing.T) {
	tests := []struct {
		name        string
		frameworkID string
		wantNil     bool
		wantID      string
	}{
		{"NIST returns SI-2", FrameworkNIST80053, false, "SI-2"},
		{"PCI returns 6.3.3", FrameworkPCIDSSv4, false, "6.3.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fw := GetFramework(tt.frameworkID)
			if fw == nil {
				t.Fatalf("framework %q not found", tt.frameworkID)
			}
			ctrl := fw.PatchSLAControl()
			if tt.wantNil {
				if ctrl != nil {
					t.Errorf("PatchSLAControl() = %v, want nil", ctrl)
				}
				return
			}
			if ctrl == nil {
				t.Fatal("PatchSLAControl() = nil, want control")
				return
			}
			if ctrl.ID != tt.wantID {
				t.Errorf("PatchSLAControl().ID = %q, want %q", ctrl.ID, tt.wantID)
			}
		})
	}

	// Test unknown framework returns nil for PatchSLAControl
	t.Run("unknown framework returns nil from GetFramework", func(t *testing.T) {
		fw := GetFramework("unknown")
		if fw != nil {
			t.Errorf("GetFramework(\"unknown\") = %v, want nil", fw)
		}
	})
}

// helpers

func intPtr(v int) *int {
	return &v
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func intPtrStr(p *int) string {
	if p == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *p)
}
