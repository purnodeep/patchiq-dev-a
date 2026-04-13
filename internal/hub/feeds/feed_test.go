package feeds

import (
	"testing"
	"time"
)

func TestRawEntryValidate(t *testing.T) {
	t.Parallel()

	validEntry := func() RawEntry {
		return RawEntry{
			CVEs:          []string{"CVE-2024-1234"},
			Name:          "KB5034441",
			Vendor:        "Microsoft",
			Product:       "Windows 11",
			Version:       "23H2",
			Severity:      "critical",
			CVSSScore:     9.8,
			OSFamily:      "windows",
			OSVersions:    []string{"11-23H2", "11-22H2"},
			InstallerType: "msu",
			ReleaseDate:   time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC),
			Summary:       "Security update for Windows 11",
			SourceURL:     "https://support.microsoft.com/kb5034441",
			Metadata:      map[string]string{"kb": "5034441"},
		}
	}

	tests := []struct {
		name    string
		entry   RawEntry
		wantErr string
	}{
		{
			name:    "valid entry with all fields",
			entry:   validEntry(),
			wantErr: "",
		},
		{
			name: "missing name",
			entry: func() RawEntry {
				e := validEntry()
				e.Name = ""
				return e
			}(),
			wantErr: "name is required",
		},
		{
			name: "missing vendor",
			entry: func() RawEntry {
				e := validEntry()
				e.Vendor = ""
				return e
			}(),
			wantErr: "vendor is required",
		},
		{
			name: "invalid severity",
			entry: func() RawEntry {
				e := validEntry()
				e.Severity = "extreme"
				return e
			}(),
			wantErr: `invalid severity "extreme"`,
		},
		{
			name: "empty severity is allowed",
			entry: func() RawEntry {
				e := validEntry()
				e.Severity = ""
				return e
			}(),
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.entry.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %q", tt.wantErr, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRawEntry_EnrichmentFields(t *testing.T) {
	t.Parallel()

	dueDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	lastMod := time.Date(2024, 2, 12, 0, 0, 0, 0, time.UTC)

	entry := RawEntry{
		Name:            "CVE-2024-21762",
		Vendor:          "fortinet",
		Severity:        "critical",
		CVSSScore:       9.8,
		CVSSv3Vector:    "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
		AttackVector:    "NETWORK",
		CweID:           "CWE-787",
		CISAKEVDueDate:  &dueDate,
		NVDLastModified: &lastMod,
		References: []CVEReference{
			{URL: "https://fortiguard.fortinet.com/psirt/FG-IR-24-015", Source: "nvd"},
			{URL: "https://nvd.nist.gov/vuln/detail/CVE-2024-21762", Source: "nvd"},
		},
	}

	if err := entry.Validate(); err != nil {
		t.Fatalf("expected valid entry, got: %v", err)
	}

	if entry.CVSSv3Vector != "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H" {
		t.Errorf("CVSSv3Vector: got %q", entry.CVSSv3Vector)
	}
	if entry.AttackVector != "NETWORK" {
		t.Errorf("AttackVector: got %q", entry.AttackVector)
	}
	if entry.CweID != "CWE-787" {
		t.Errorf("CweID: got %q", entry.CweID)
	}
	if !entry.CISAKEVDueDate.Equal(dueDate) {
		t.Errorf("CISAKEVDueDate: got %v", entry.CISAKEVDueDate)
	}
	if len(entry.References) != 2 {
		t.Errorf("References: expected 2, got %d", len(entry.References))
	}
	if !entry.NVDLastModified.Equal(lastMod) {
		t.Errorf("NVDLastModified: got %v", entry.NVDLastModified)
	}
}
