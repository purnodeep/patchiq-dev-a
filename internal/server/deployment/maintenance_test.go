package deployment

import (
	"testing"
	"time"
)

func TestIsInMaintenanceWindow(t *testing.T) {
	tests := []struct {
		name   string
		window *MaintenanceWindow
		now    time.Time
		want   bool
	}{
		{
			name:   "nil window means always open",
			window: nil,
			now:    time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
			want:   true,
		},
		{
			name:   "within window",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 3, 0, 0, 0, time.UTC), // Saturday 03:00 UTC
			want:   true,
		},
		{
			name:   "outside window wrong day",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Sunday}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 3, 0, 0, 0, time.UTC), // Saturday
			want:   false,
		},
		{
			name:   "outside window wrong time",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC), // Saturday 10:00
			want:   false,
		},
		{
			name:   "timezone aware",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "06:00", TZ: "America/New_York"},
			now:    time.Date(2026, 3, 7, 8, 0, 0, 0, time.UTC), // Saturday 03:00 ET (EST = UTC-5)
			want:   true,
		},
		{
			name:   "at exact start time is in window",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 2, 0, 0, 0, time.UTC),
			want:   true,
		},
		{
			name:   "at exact end time is outside window",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 6, 0, 0, 0, time.UTC),
			want:   false,
		},
		{
			name:   "empty days means always open",
			window: &MaintenanceWindow{Days: []time.Weekday{}, Start: "02:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
			want:   true,
		},
		{
			name:   "overnight window inside at 23:00",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "22:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 23, 0, 0, 0, time.UTC), // Saturday 23:00
			want:   true,
		},
		{
			name:   "overnight window inside at 02:00 next day (cross-day)",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "22:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 8, 2, 0, 0, 0, time.UTC), // Sunday 02:00 (Saturday's window spills over)
			want:   true,
		},
		{
			name:   "overnight window outside at 07:00 next day",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "22:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC), // Sunday 07:00 (past the window end)
			want:   false,
		},
		{
			name:   "overnight window outside at 21:00",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "22:00", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 21, 0, 0, 0, time.UTC), // Saturday 21:00
			want:   false,
		},
		{
			name:   "invalid start time returns false",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "bad", End: "06:00", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 3, 0, 0, 0, time.UTC),
			want:   false,
		},
		{
			name:   "invalid end time returns false",
			window: &MaintenanceWindow{Days: []time.Weekday{time.Saturday}, Start: "02:00", End: "nope", TZ: "UTC"},
			now:    time.Date(2026, 3, 7, 3, 0, 0, 0, time.UTC),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInMaintenanceWindow(tt.window, tt.now)
			if got != tt.want {
				t.Errorf("IsInMaintenanceWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseHHMM(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantH   int
		wantM   int
		wantErr bool
	}{
		{"valid", "02:30", 2, 30, false},
		{"midnight", "00:00", 0, 0, false},
		{"end of day", "23:59", 23, 59, false},
		{"invalid format", "abc", 0, 0, true},
		{"empty string", "", 0, 0, true},
		{"partial", "12", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m, err := parseHHMM(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHHMM(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr {
				if h != tt.wantH || m != tt.wantM {
					t.Errorf("parseHHMM(%q) = (%d, %d), want (%d, %d)", tt.input, h, m, tt.wantH, tt.wantM)
				}
			}
		})
	}
}

func TestParseMaintenanceWindow(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantNil bool
		wantErr bool
	}{
		{name: "nil data returns nil", data: nil, wantNil: true},
		{name: "empty data returns nil", data: []byte{}, wantNil: true},
		{name: "valid json", data: []byte(`{"days":[1,2,3,4,5],"start":"02:00","end":"06:00","tz":"UTC"}`), wantNil: false},
		{name: "invalid json", data: []byte(`{invalid}`), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := ParseMaintenanceWindow(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseMaintenanceWindow() error = %v, wantErr %v", err, tt.wantErr)
			}
			if (mw == nil) != tt.wantNil && !tt.wantErr {
				t.Errorf("ParseMaintenanceWindow() nil = %v, wantNil %v", mw == nil, tt.wantNil)
			}
		})
	}
}
