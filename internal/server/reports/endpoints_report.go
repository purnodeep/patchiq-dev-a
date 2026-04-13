package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// EndpointsQuerier defines the database methods needed by the endpoints assembler.
type EndpointsQuerier interface {
	ListEndpoints(ctx context.Context, arg sqlcgen.ListEndpointsParams) ([]sqlcgen.ListEndpointsRow, error)
	GetDashboardSummary(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetDashboardSummaryRow, error)
	GetEndpointOsSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetEndpointOsSummaryRow, error)
	GetEndpointStatusSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetEndpointStatusSummaryRow, error)
}

// EndpointsAssembler builds a ReportData for the endpoints report.
type EndpointsAssembler struct {
	q EndpointsQuerier
}

// NewEndpointsAssembler creates a new EndpointsAssembler.
func NewEndpointsAssembler(q EndpointsQuerier) *EndpointsAssembler {
	return &EndpointsAssembler{q: q}
}

// Assemble fetches endpoint data and builds a ReportData.
func (a *EndpointsAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble endpoints report: parse tenant id: %w", err)
	}

	tagID := pgtype.UUID{}
	if opts.Filters.TagID != "" {
		if err := tagID.Scan(opts.Filters.TagID); err != nil {
			return nil, fmt.Errorf("assemble endpoints report: parse tag id: %w", err)
		}
	}

	// Fetch all endpoints (use large page limit, no cursor).
	endpoints, err := a.q.ListEndpoints(ctx, sqlcgen.ListEndpointsParams{
		TenantID:  tid,
		Status:    opts.Filters.Status,
		OsFamily:  opts.Filters.OSFamily,
		Search:    "",
		TagID:     tagID,
		PageLimit: 10000,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble endpoints report: list endpoints: %w", err)
	}

	summary, err := a.q.GetDashboardSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble endpoints report: get dashboard summary: %w", err)
	}

	osSummary, err := a.q.GetEndpointOsSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble endpoints report: get os summary: %w", err)
	}

	statusSummary, err := a.q.GetEndpointStatusSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble endpoints report: get status summary: %w", err)
	}

	now := time.Now().In(IST)

	// Build subtitle from filters.
	var subtitleParts []string
	if opts.Filters.Status != "" {
		subtitleParts = append(subtitleParts, "Status: "+opts.Filters.Status)
	}
	if opts.Filters.OSFamily != "" {
		subtitleParts = append(subtitleParts, "OS: "+opts.Filters.OSFamily)
	}
	subtitle := strings.Join(subtitleParts, " | ")

	// Parse date filters.
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	// Summary stats.
	onlinePct := float64(0)
	if summary.EndpointsTotal > 0 {
		onlinePct = float64(summary.EndpointsOnline) * 100 / float64(summary.EndpointsTotal)
	}

	stats := []StatBox{
		{Label: "Total Endpoints", Value: fmt.Sprintf("%d", summary.EndpointsTotal), Color: "blue"},
		{Label: "Online", Value: fmt.Sprintf("%d (%.1f%%)", summary.EndpointsOnline, onlinePct), Color: "green"},
		{Label: "Compliance", Value: fmt.Sprintf("%.1f%%", summary.CompliancePct), Color: scoreColorName(summary.CompliancePct)},
		{Label: "Critical CVEs", Value: fmt.Sprintf("%d", summary.CvesCritical), Color: "red"},
		{Label: "Pending Patches", Value: fmt.Sprintf("%d", summary.PatchesAvailable), Color: "orange"},
	}

	// Charts.
	osChart := ChartSpec{
		Title:  "OS Distribution",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	osColors := map[string]string{
		"windows": ColorBlue,
		"linux":   ColorGreen,
		"macos":   ColorOrange,
		"darwin":  ColorOrange,
		"unknown": ColorGray,
	}
	for _, os := range osSummary {
		color := osColors[strings.ToLower(os.OsFamily)]
		if color == "" {
			color = ColorGray
		}
		osChart.Data = append(osChart.Data, ChartDataPoint{
			Label: os.OsFamily,
			Value: float64(os.Count),
			Color: color,
		})
	}

	statusChart := ChartSpec{
		Title:  "Status Distribution",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	statusColors := map[string]string{
		"online":             ColorGreen,
		"offline":            ColorRed,
		"degraded":           ColorOrange,
		"decommissioned":     ColorGray,
		"pending_enrollment": ColorYellow,
	}
	for _, s := range statusSummary {
		color := statusColors[s.Status]
		if color == "" {
			color = ColorGray
		}
		statusChart.Data = append(statusChart.Data, ChartDataPoint{
			Label: s.Status,
			Value: float64(s.Count),
			Color: color,
		})
	}

	// Breakdown by OS Family.
	type osBreakdown struct {
		count     int
		online    int
		cves      int64
		patches   int64
		compTotal float64
		compCount int
	}
	osBD := make(map[string]*osBreakdown)
	for _, ep := range endpoints {
		bd, ok := osBD[ep.OsFamily]
		if !ok {
			bd = &osBreakdown{}
			osBD[ep.OsFamily] = bd
		}
		bd.count++
		if ep.Status == "online" {
			bd.online++
		}
		bd.cves += ep.CriticalCveCount
		bd.patches += ep.PendingPatchesCount
		if ep.CompliancePct.Valid {
			bd.compTotal += ep.CompliancePct.Float64
			bd.compCount++
		}
	}
	bdTable := BreakdownTable{
		Title:   "By OS Family",
		Columns: []string{"OS Family", "Count", "Online", "Critical CVEs", "Pending Patches", "Compliance %"},
	}
	for osName, bd := range osBD {
		compPct := float64(0)
		if bd.compCount > 0 {
			compPct = bd.compTotal / float64(bd.compCount)
		}
		bdTable.Rows = append(bdTable.Rows, []string{
			osName,
			fmt.Sprintf("%d", bd.count),
			fmt.Sprintf("%d", bd.online),
			fmt.Sprintf("%d", bd.cves),
			fmt.Sprintf("%d", bd.patches),
			fmt.Sprintf("%.1f%%", compPct),
		})
	}

	// Highlights: high-risk endpoints (risk_score >= 7).
	highlights := HighlightSection{
		Title:       "High Risk Endpoints",
		Description: "Endpoints with risk score >= 7 require immediate attention",
		Columns:     []string{"Hostname", "OS", "Risk Score", "Critical CVEs", "Pending Patches"},
	}
	for _, ep := range endpoints {
		riskScore := calcRiskScore(ep.CriticalCveCount, ep.HighCveCount, ep.MediumCveCount)
		if riskScore >= 7 {
			highlights.Rows = append(highlights.Rows, []string{
				ep.Hostname,
				ep.OsFamily,
				fmt.Sprintf("%.1f", riskScore),
				fmt.Sprintf("%d", ep.CriticalCveCount),
				fmt.Sprintf("%d", ep.PendingPatchesCount),
			})
			highlights.RowColors = append(highlights.RowColors, ColorRed)
		}
	}

	// Detail table.
	detail := DetailTable{
		Columns: []string{
			"Hostname", "OS", "Version", "Status", "Agent Version", "IP",
			"Risk Score", "Critical CVEs", "High CVEs", "Pending Patches",
			"Compliance %", "Tags", "Last Seen",
		},
		MaxPDFRows: 500,
		TotalRows:  len(endpoints),
	}
	for i, ep := range endpoints {
		if i >= 500 {
			break
		}
		riskScore := calcRiskScore(ep.CriticalCveCount, ep.HighCveCount, ep.MediumCveCount)
		compStr := "N/A"
		if ep.CompliancePct.Valid {
			compStr = fmt.Sprintf("%.1f%%", ep.CompliancePct.Float64)
		}

		lastSeen := ""
		if ep.LastSeen.Valid {
			lastSeen = ep.LastSeen.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}

		detail.Rows = append(detail.Rows, []string{
			ep.Hostname,
			ep.OsFamily,
			ep.OsVersion,
			ep.Status,
			pgTextStr(ep.AgentVersion),
			pgTextStr(ep.IpAddress),
			fmt.Sprintf("%.1f", riskScore),
			fmt.Sprintf("%d", ep.CriticalCveCount),
			fmt.Sprintf("%d", ep.HighCveCount),
			fmt.Sprintf("%d", ep.PendingPatchesCount),
			compStr,
			formatTags(ep.Tags),
			lastSeen,
		})

		switch {
		case riskScore >= 7:
			detail.RowColors = append(detail.RowColors, ColorRed)
		case riskScore >= 4:
			detail.RowColors = append(detail.RowColors, ColorOrange)
		default:
			detail.RowColors = append(detail.RowColors, ColorGreen)
		}
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "Endpoints Report",
			Subtitle:    subtitle,
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{osChart, statusChart},
		Breakdowns: []BreakdownTable{bdTable},
		Highlights: []HighlightSection{highlights},
		Detail:     detail,
	}, nil
}

func calcRiskScore(critical, high, medium int64) float64 {
	score := float64(critical*3+high*2+medium) / 10.0
	return math.Min(score, 10.0)
}

func pgTextStr(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

func formatTags(tags interface{}) string {
	if tags == nil {
		return ""
	}
	raw, err := json.Marshal(tags)
	if err != nil {
		return ""
	}
	var tagList []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &tagList); err != nil {
		return ""
	}
	names := make([]string, 0, len(tagList))
	for _, t := range tagList {
		names = append(names, t.Name)
	}
	return strings.Join(names, ", ")
}

func parseDateFilters(from, to string) (time.Time, time.Time) {
	var dateFrom, dateTo time.Time
	if from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			dateFrom = t.In(IST)
		}
	}
	if to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			dateTo = t.In(IST)
		}
	}
	if dateFrom.IsZero() {
		dateFrom = time.Now().In(IST).AddDate(0, -1, 0)
	}
	if dateTo.IsZero() {
		dateTo = time.Now().In(IST)
	}
	return dateFrom, dateTo
}

func scoreColorName(pct float64) string {
	switch {
	case pct < 50:
		return "red"
	case pct < 75:
		return "orange"
	case pct < 90:
		return "yellow"
	default:
		return "green"
	}
}
