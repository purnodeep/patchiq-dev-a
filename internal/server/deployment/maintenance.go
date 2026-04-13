package deployment

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// MaintenanceWindow defines when an endpoint is available for patching.
type MaintenanceWindow struct {
	Days  []time.Weekday `json:"days"`
	Start string         `json:"start"` // "HH:MM" format
	End   string         `json:"end"`   // "HH:MM" format
	TZ    string         `json:"tz"`    // IANA timezone
}

// ParseMaintenanceWindow parses JSONB maintenance window data from the database.
// Returns nil if data is nil or empty (meaning "always available").
func ParseMaintenanceWindow(data []byte) (*MaintenanceWindow, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var mw MaintenanceWindow
	if err := json.Unmarshal(data, &mw); err != nil {
		return nil, fmt.Errorf("parse maintenance window: %w", err)
	}
	return &mw, nil
}

// IsInMaintenanceWindow checks if the given time falls within the maintenance window.
// Returns true if window is nil or has no days configured (no restriction).
func IsInMaintenanceWindow(mw *MaintenanceWindow, now time.Time) bool {
	if mw == nil || len(mw.Days) == 0 {
		return true
	}

	loc, err := time.LoadLocation(mw.TZ)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)

	startH, startM, err := parseHHMM(mw.Start)
	if err != nil {
		slog.Error("maintenance window: invalid start time, treating as outside window",
			"start", mw.Start, "error", err)
		return false
	}
	endH, endM, err := parseHHMM(mw.End)
	if err != nil {
		slog.Error("maintenance window: invalid end time, treating as outside window",
			"end", mw.End, "error", err)
		return false
	}

	minuteOfDay := localNow.Hour()*60 + localNow.Minute()
	startMinute := startH*60 + startM
	endMinute := endH*60 + endM

	isOvernight := startMinute > endMinute

	// For overnight windows (e.g., 22:00-06:00), the after-midnight portion
	// belongs to the previous day's window. Check yesterday's weekday instead.
	checkDay := localNow.Weekday()
	if isOvernight && minuteOfDay < endMinute {
		checkDay = localNow.AddDate(0, 0, -1).Weekday()
	}

	dayMatch := false
	for _, d := range mw.Days {
		if checkDay == d {
			dayMatch = true
			break
		}
	}
	if !dayMatch {
		return false
	}

	if !isOvernight {
		return minuteOfDay >= startMinute && minuteOfDay < endMinute
	}
	return minuteOfDay >= startMinute || minuteOfDay < endMinute
}

func parseHHMM(s string) (int, int, error) {
	var h, m int
	n, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return 0, 0, fmt.Errorf("parse maintenance time %q: expected HH:MM format", s)
	}
	return h, m, nil
}
