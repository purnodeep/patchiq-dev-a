package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// ExecutiveQuerier defines the database methods needed by the executive assembler.
type ExecutiveQuerier interface {
	GetDashboardSummary(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetDashboardSummaryRow, error)
	GetOverallComplianceScore(ctx context.Context, tenantID pgtype.UUID) (sqlcgen.GetOverallComplianceScoreRow, error)
	GetFrameworkScoreSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetFrameworkScoreSummaryRow, error)
	CountCVEsBySeverity(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountCVEsBySeverityRow, error)
	GetEndpointStatusSummary(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetEndpointStatusSummaryRow, error)
	GetTopEndpointsByRisk(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.GetTopEndpointsByRiskRow, error)
	CountCVEsKEV(ctx context.Context, tenantID pgtype.UUID) (int32, error)
	ListPatchesFiltered(ctx context.Context, arg sqlcgen.ListPatchesFilteredParams) ([]sqlcgen.ListPatchesFilteredRow, error)
}

// ExecutiveAssembler builds a ReportData for the executive summary report.
type ExecutiveAssembler struct {
	q ExecutiveQuerier
}

// NewExecutiveAssembler creates a new ExecutiveAssembler.
func NewExecutiveAssembler(q ExecutiveQuerier) *ExecutiveAssembler {
	return &ExecutiveAssembler{q: q}
}

// Assemble fetches summary data and builds an executive ReportData.
func (a *ExecutiveAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble executive report: parse tenant id: %w", err)
	}

	dashboard, err := a.q.GetDashboardSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: get dashboard summary: %w", err)
	}

	compliance, err := a.q.GetOverallComplianceScore(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: get compliance score: %w", err)
	}

	frameworkScores, err := a.q.GetFrameworkScoreSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: get framework scores: %w", err)
	}

	sevCounts, err := a.q.CountCVEsBySeverity(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: count cves by severity: %w", err)
	}

	statusSummary, err := a.q.GetEndpointStatusSummary(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: get endpoint status: %w", err)
	}

	topRisk, err := a.q.GetTopEndpointsByRisk(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: get top risk endpoints: %w", err)
	}

	kevCount, err := a.q.CountCVEsKEV(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: count kev: %w", err)
	}

	// Get critical patches with low remediation for highlights.
	criticalPatches, err := a.q.ListPatchesFiltered(ctx, sqlcgen.ListPatchesFilteredParams{
		TenantID:  tid,
		Severity:  "critical",
		SortBy:    "severity",
		SortDir:   "asc",
		PageLimit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble executive report: list critical patches: %w", err)
	}

	now := time.Now().In(IST)
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	// Overall compliance score.
	overallScore := numericToFloat(compliance.OverallScore)

	// Deployment success rate from dashboard.
	onlinePct := float64(0)
	if dashboard.EndpointsTotal > 0 {
		onlinePct = float64(dashboard.EndpointsOnline) * 100 / float64(dashboard.EndpointsTotal)
	}

	// Risk score: average of top risk endpoints.
	avgRisk := float64(0)
	if len(topRisk) > 0 {
		var riskSum int32
		for _, r := range topRisk {
			riskSum += r.RiskScore
		}
		avgRisk = float64(riskSum) / float64(len(topRisk))
	}

	// Deployment success rate.
	totalDeployments := int32(dashboard.DeploymentsCompletedToday) + dashboard.FailedDeploymentsCount + dashboard.DeploymentsRunning
	deploySuccessRate := float64(0)
	if totalDeployments > 0 {
		deploySuccessRate = float64(dashboard.DeploymentsCompletedToday) * 100 / float64(totalDeployments)
	}

	stats := []StatBox{
		{Label: "Overall Compliance", Value: fmt.Sprintf("%.1f%%", overallScore), Color: scoreColorName(overallScore)},
		{Label: "Endpoints", Value: fmt.Sprintf("%d (%.1f%% online)", dashboard.EndpointsTotal, onlinePct), Color: "blue"},
		{Label: "Critical CVEs", Value: fmt.Sprintf("%d", dashboard.CvesCritical), Color: "red"},
		{Label: "Deployment Success Rate", Value: fmt.Sprintf("%.1f%%", deploySuccessRate), Color: scoreColorName(deploySuccessRate)},
		{Label: "Avg Risk Score", Value: fmt.Sprintf("%.1f/10", avgRisk), Color: "orange"},
	}

	// Chart: compliance by framework (bar).
	fwChart := ChartSpec{
		Title:  "Compliance by Framework",
		Type:   "bar",
		Width:  600,
		Height: 300,
	}
	for _, fw := range frameworkScores {
		fwScore := numericToFloat(fw.Score)
		fwChart.Data = append(fwChart.Data, ChartDataPoint{
			Label: fw.FrameworkID,
			Value: fwScore,
			Color: ScoreColor(fwScore),
		})
	}

	// Chart: CVE severity pie.
	sevChart := ChartSpec{
		Title:  "CVE Severity Distribution",
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

	// Chart: endpoint status donut.
	statusChart := ChartSpec{
		Title:  "Endpoint Status",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	epStatusColors := map[string]string{
		"online":             ColorGreen,
		"offline":            ColorRed,
		"degraded":           ColorOrange,
		"decommissioned":     ColorGray,
		"pending_enrollment": ColorYellow,
	}
	for _, s := range statusSummary {
		color := epStatusColors[s.Status]
		if color == "" {
			color = ColorGray
		}
		statusChart.Data = append(statusChart.Data, ChartDataPoint{
			Label: s.Status,
			Value: float64(s.Count),
			Color: color,
		})
	}

	// Breakdown: framework scores table.
	fwBreakdown := BreakdownTable{
		Title:   "Framework Scores",
		Columns: []string{"Framework", "Score"},
	}
	for _, fw := range frameworkScores {
		fwScore := numericToFloat(fw.Score)
		fwBreakdown.Rows = append(fwBreakdown.Rows, []string{
			fw.FrameworkID,
			fmt.Sprintf("%.1f%%", fwScore),
		})
		fwBreakdown.RowColors = append(fwBreakdown.RowColors, ScoreColor(fwScore))
	}

	// Breakdown: top 5 endpoints by risk.
	riskBreakdown := BreakdownTable{
		Title:   "Top Endpoints by Risk",
		Columns: []string{"Hostname", "CVE Count", "Risk Score"},
	}
	limit := 5
	if len(topRisk) < limit {
		limit = len(topRisk)
	}
	for _, r := range topRisk[:limit] {
		riskBreakdown.Rows = append(riskBreakdown.Rows, []string{
			r.Hostname,
			fmt.Sprintf("%d", r.CveCount),
			fmt.Sprintf("%d", r.RiskScore),
		})
		switch {
		case r.RiskScore >= 7:
			riskBreakdown.RowColors = append(riskBreakdown.RowColors, ColorRed)
		case r.RiskScore >= 4:
			riskBreakdown.RowColors = append(riskBreakdown.RowColors, ColorOrange)
		default:
			riskBreakdown.RowColors = append(riskBreakdown.RowColors, ColorGreen)
		}
	}

	// Highlights.
	var highlightSections []HighlightSection

	// CISA KEV overdue highlight.
	if kevCount > 0 {
		highlightSections = append(highlightSections, HighlightSection{
			Title:       "CISA KEV Vulnerabilities",
			Description: fmt.Sprintf("%d CVEs are listed in the CISA Known Exploited Vulnerabilities catalog", kevCount),
			Columns:     []string{"Metric", "Value"},
			Rows:        [][]string{{"CISA KEV CVEs", fmt.Sprintf("%d", kevCount)}},
			RowColors:   []string{ColorRed},
		})
	}

	// Critical patches with low remediation.
	lowRemPatches := HighlightSection{
		Title:       "Critical Patches with Low Remediation",
		Description: "Critical patches with less than 25% remediation",
		Columns:     []string{"Name", "Remediation %", "Affected Endpoints"},
	}
	for _, p := range criticalPatches {
		if p.RemediationPct < 25 {
			lowRemPatches.Rows = append(lowRemPatches.Rows, []string{
				p.Name,
				fmt.Sprintf("%d%%", p.RemediationPct),
				fmt.Sprintf("%d", p.AffectedEndpointCount),
			})
			lowRemPatches.RowColors = append(lowRemPatches.RowColors, ColorRed)
		}
	}
	if len(lowRemPatches.Rows) > 0 {
		highlightSections = append(highlightSections, lowRemPatches)
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "Executive Summary",
			Subtitle:    "Organization-wide security posture overview",
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{fwChart, sevChart, statusChart},
		Breakdowns: []BreakdownTable{fwBreakdown, riskBreakdown},
		Highlights: highlightSections,
		Detail:     DetailTable{}, // Executive summary has no detail table.
	}, nil
}
