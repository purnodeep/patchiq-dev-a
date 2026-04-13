package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// PatchesQuerier defines the database methods needed by the patches assembler.
type PatchesQuerier interface {
	ListPatchesFiltered(ctx context.Context, arg sqlcgen.ListPatchesFilteredParams) ([]sqlcgen.ListPatchesFilteredRow, error)
	CountPatchesFiltered(ctx context.Context, arg sqlcgen.CountPatchesFilteredParams) (int64, error)
	CountPatchesBySeverity(ctx context.Context, arg sqlcgen.CountPatchesBySeverityParams) ([]sqlcgen.CountPatchesBySeverityRow, error)
}

// PatchesAssembler builds a ReportData for the patches report.
type PatchesAssembler struct {
	q PatchesQuerier
}

// NewPatchesAssembler creates a new PatchesAssembler.
func NewPatchesAssembler(q PatchesQuerier) *PatchesAssembler {
	return &PatchesAssembler{q: q}
}

// Assemble fetches patch data and builds a ReportData.
func (a *PatchesAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble patches report: parse tenant id: %w", err)
	}

	patches, err := a.q.ListPatchesFiltered(ctx, sqlcgen.ListPatchesFilteredParams{
		TenantID:  tid,
		Severity:  opts.Filters.Severity,
		OsFamily:  opts.Filters.OSFamily,
		Status:    opts.Filters.Status,
		SortBy:    "severity",
		SortDir:   "asc",
		PageLimit: 10000,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble patches report: list patches: %w", err)
	}

	totalCount, err := a.q.CountPatchesFiltered(ctx, sqlcgen.CountPatchesFilteredParams{
		TenantID: tid,
		Severity: opts.Filters.Severity,
		OsFamily: opts.Filters.OSFamily,
		Status:   opts.Filters.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble patches report: count patches: %w", err)
	}

	sevCounts, err := a.q.CountPatchesBySeverity(ctx, sqlcgen.CountPatchesBySeverityParams{
		TenantID: tid,
		OsFamily: opts.Filters.OSFamily,
		Status:   opts.Filters.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble patches report: count by severity: %w", err)
	}

	now := time.Now().In(IST)
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	// Aggregate stats.
	var criticalCount, highCount, zeroRemediation int32
	var remediationSum int64
	for _, p := range patches {
		switch p.Severity {
		case "critical":
			criticalCount++
		case "high":
			highCount++
		}
		remediationSum += int64(p.RemediationPct)
		if p.RemediationPct == 0 {
			zeroRemediation++
		}
	}
	avgRemediation := float64(0)
	if len(patches) > 0 {
		avgRemediation = float64(remediationSum) / float64(len(patches))
	}

	// Build subtitle.
	var subtitleParts []string
	if opts.Filters.Severity != "" {
		subtitleParts = append(subtitleParts, "Severity: "+opts.Filters.Severity)
	}
	if opts.Filters.OSFamily != "" {
		subtitleParts = append(subtitleParts, "OS: "+opts.Filters.OSFamily)
	}
	subtitle := strings.Join(subtitleParts, " | ")

	stats := []StatBox{
		{Label: "Total Patches", Value: fmt.Sprintf("%d", totalCount), Color: "blue"},
		{Label: "Critical", Value: fmt.Sprintf("%d", criticalCount), Color: "red"},
		{Label: "High", Value: fmt.Sprintf("%d", highCount), Color: "orange"},
		{Label: "Avg Remediation", Value: fmt.Sprintf("%.1f%%", avgRemediation), Color: scoreColorName(avgRemediation)},
		{Label: "Zero Remediation", Value: fmt.Sprintf("%d", zeroRemediation), Color: "red"},
	}

	// Severity distribution chart.
	sevChart := ChartSpec{
		Title:  "Severity Distribution",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	for _, sc := range sevCounts {
		sevChart.Data = append(sevChart.Data, ChartDataPoint{
			Label: sc.Severity,
			Value: float64(sc.Count),
			Color: SeverityColor(sc.Severity),
		})
	}

	// Breakdown by severity.
	sevBreakdown := BreakdownTable{
		Title:   "By Severity",
		Columns: []string{"Severity", "Count", "Avg Remediation %"},
	}
	type sevAgg struct {
		count    int
		remTotal int64
	}
	sevMap := make(map[string]*sevAgg)
	for _, p := range patches {
		sa, ok := sevMap[p.Severity]
		if !ok {
			sa = &sevAgg{}
			sevMap[p.Severity] = sa
		}
		sa.count++
		sa.remTotal += int64(p.RemediationPct)
	}
	for _, sev := range []string{"critical", "high", "medium", "low"} {
		sa := sevMap[sev]
		if sa == nil {
			continue
		}
		avg := float64(sa.remTotal) / float64(sa.count)
		sevBreakdown.Rows = append(sevBreakdown.Rows, []string{
			sev,
			fmt.Sprintf("%d", sa.count),
			fmt.Sprintf("%.1f%%", avg),
		})
		sevBreakdown.RowColors = append(sevBreakdown.RowColors, SeverityColor(sev))
	}

	// Breakdown by OS family.
	osBreakdown := BreakdownTable{
		Title:   "By OS Family",
		Columns: []string{"OS Family", "Count", "Critical", "Avg Remediation %"},
	}
	type osAgg struct {
		count    int
		critical int
		remTotal int64
	}
	osMap := make(map[string]*osAgg)
	for _, p := range patches {
		oa, ok := osMap[p.OsFamily]
		if !ok {
			oa = &osAgg{}
			osMap[p.OsFamily] = oa
		}
		oa.count++
		if p.Severity == "critical" {
			oa.critical++
		}
		oa.remTotal += int64(p.RemediationPct)
	}
	for osName, oa := range osMap {
		avg := float64(oa.remTotal) / float64(oa.count)
		osBreakdown.Rows = append(osBreakdown.Rows, []string{
			osName,
			fmt.Sprintf("%d", oa.count),
			fmt.Sprintf("%d", oa.critical),
			fmt.Sprintf("%.1f%%", avg),
		})
	}

	// Highlights: critical/high patches with low remediation.
	highlights := HighlightSection{
		Title:       "Low Remediation Patches",
		Description: "Critical and high severity patches with less than 50% remediation",
		Columns:     []string{"Name", "Severity", "Remediation %", "Affected Endpoints"},
	}
	for _, p := range patches {
		if (p.Severity == "critical" || p.Severity == "high") && p.RemediationPct < 50 {
			highlights.Rows = append(highlights.Rows, []string{
				p.Name,
				p.Severity,
				fmt.Sprintf("%d%%", p.RemediationPct),
				fmt.Sprintf("%d", p.AffectedEndpointCount),
			})
			highlights.RowColors = append(highlights.RowColors, SeverityColor(p.Severity))
		}
	}

	// Detail table.
	detail := DetailTable{
		Columns: []string{
			"Name", "Version", "Severity", "OS", "CVE Count", "Highest CVSS",
			"Affected Endpoints", "Deployed", "Remediation %", "Released", "Status",
		},
		MaxPDFRows: 500,
		TotalRows:  int(totalCount),
	}
	for i, p := range patches {
		if i >= 500 {
			break
		}
		released := ""
		if p.ReleasedAt.Valid {
			released = p.ReleasedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}
		detail.Rows = append(detail.Rows, []string{
			p.Name,
			p.Version,
			p.Severity,
			p.OsFamily,
			fmt.Sprintf("%d", p.CveCount),
			fmt.Sprintf("%.1f", p.HighestCvssScore),
			fmt.Sprintf("%d", p.AffectedEndpointCount),
			fmt.Sprintf("%d", p.EndpointsDeployedCount),
			fmt.Sprintf("%d%%", p.RemediationPct),
			released,
			p.Status,
		})
		detail.RowColors = append(detail.RowColors, SeverityColor(p.Severity))
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "Patches Report",
			Subtitle:    subtitle,
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{sevChart},
		Breakdowns: []BreakdownTable{sevBreakdown, osBreakdown},
		Highlights: []HighlightSection{highlights},
		Detail:     detail,
	}, nil
}
