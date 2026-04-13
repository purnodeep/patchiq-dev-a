package reports

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// ComplianceQuerier defines the database methods needed by the compliance assembler.
type ComplianceQuerier interface {
	GetLatestFrameworkScore(ctx context.Context, arg sqlcgen.GetLatestFrameworkScoreParams) (sqlcgen.ComplianceScore, error)
	ListControlResultsByFramework(ctx context.Context, arg sqlcgen.ListControlResultsByFrameworkParams) ([]sqlcgen.ComplianceControlResult, error)
	ListScoreTrend(ctx context.Context, arg sqlcgen.ListScoreTrendParams) ([]sqlcgen.ComplianceScore, error)
	ListNonCompliantEndpointsByFramework(ctx context.Context, arg sqlcgen.ListNonCompliantEndpointsByFrameworkParams) ([]sqlcgen.ListNonCompliantEndpointsByFrameworkRow, error)
	ListOverdueControls(ctx context.Context, arg sqlcgen.ListOverdueControlsParams) ([]sqlcgen.ComplianceControlResult, error)
}

// ComplianceAssembler builds a ReportData for the compliance report.
type ComplianceAssembler struct {
	q ComplianceQuerier
}

// NewComplianceAssembler creates a new ComplianceAssembler.
func NewComplianceAssembler(q ComplianceQuerier) *ComplianceAssembler {
	return &ComplianceAssembler{q: q}
}

// Assemble fetches compliance data for a specific framework and builds a ReportData.
func (a *ComplianceAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble compliance report: parse tenant id: %w", err)
	}

	frameworkID := opts.Filters.FrameworkID
	if frameworkID == "" {
		return nil, fmt.Errorf("assemble compliance report: framework_id filter is required")
	}

	// Get latest framework score.
	score, err := a.q.GetLatestFrameworkScore(ctx, sqlcgen.GetLatestFrameworkScoreParams{
		TenantID:    tid,
		FrameworkID: frameworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compliance report: get framework score: %w", err)
	}

	// Get control results.
	controls, err := a.q.ListControlResultsByFramework(ctx, sqlcgen.ListControlResultsByFrameworkParams{
		TenantID:    tid,
		FrameworkID: frameworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compliance report: list controls: %w", err)
	}

	// Get score trend.
	trend, err := a.q.ListScoreTrend(ctx, sqlcgen.ListScoreTrendParams{
		TenantID:    tid,
		FrameworkID: frameworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compliance report: list score trend: %w", err)
	}

	// Get non-compliant endpoints.
	ncEndpoints, err := a.q.ListNonCompliantEndpointsByFramework(ctx, sqlcgen.ListNonCompliantEndpointsByFrameworkParams{
		TenantID:    tid,
		FrameworkID: frameworkID,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compliance report: list non-compliant endpoints: %w", err)
	}

	// Get overdue controls.
	overdueControls, err := a.q.ListOverdueControls(ctx, sqlcgen.ListOverdueControlsParams{
		TenantID:    tid,
		ResultLimit: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble compliance report: list overdue controls: %w", err)
	}
	// Filter overdue controls to this framework.
	var frameworkOverdue []sqlcgen.ComplianceControlResult
	for _, c := range overdueControls {
		if c.FrameworkID == frameworkID {
			frameworkOverdue = append(frameworkOverdue, c)
		}
	}

	now := time.Now().In(IST)
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	scorePct := numericToFloat(score.Score)

	// Count controls by status.
	var passing, failing, partial, na int
	var maxEndpoints int32
	for _, c := range controls {
		switch c.Status {
		case "pass":
			passing++
		case "fail":
			failing++
		case "partial":
			partial++
		case "na":
			na++
		}
		if c.TotalEndpoints > maxEndpoints {
			maxEndpoints = c.TotalEndpoints
		}
	}

	stats := []StatBox{
		{Label: "Framework", Value: frameworkID, Color: "blue"},
		{Label: "Score", Value: fmt.Sprintf("%.1f%%", scorePct), Color: scoreColorName(scorePct)},
		{Label: "Controls Passing", Value: fmt.Sprintf("%d/%d", passing, len(controls)), Color: "green"},
		{Label: "Endpoints Evaluated", Value: fmt.Sprintf("%d", maxEndpoints), Color: "blue"},
		{Label: "Overdue Controls", Value: fmt.Sprintf("%d", len(frameworkOverdue)), Color: "red"},
	}

	// Control status pie chart.
	controlChart := ChartSpec{
		Title:  "Control Status",
		Type:   "pie",
		Width:  400,
		Height: 300,
		Data: []ChartDataPoint{
			{Label: "Pass", Value: float64(passing), Color: ColorGreen},
			{Label: "Fail", Value: float64(failing), Color: ColorRed},
			{Label: "Partial", Value: float64(partial), Color: ColorOrange},
			{Label: "N/A", Value: float64(na), Color: ColorGray},
		},
	}

	// Score trend line chart.
	trendChart := ChartSpec{
		Title:  "Score Trend",
		Type:   "line",
		Width:  600,
		Height: 300,
	}
	for _, t := range trend {
		label := ""
		if t.EvaluatedAt.Valid {
			label = t.EvaluatedAt.Time.In(IST).Format("02 Jan")
		}
		trendChart.Data = append(trendChart.Data, ChartDataPoint{
			Label: label,
			Value: numericToFloat(t.Score),
			Color: ColorBlue,
		})
	}

	// Breakdown: control results by category.
	type catAgg struct {
		pass, fail, partial, na int
	}
	catMap := make(map[string]*catAgg)
	for _, c := range controls {
		ca, ok := catMap[c.Category]
		if !ok {
			ca = &catAgg{}
			catMap[c.Category] = ca
		}
		switch c.Status {
		case "pass":
			ca.pass++
		case "fail":
			ca.fail++
		case "partial":
			ca.partial++
		case "na":
			ca.na++
		}
	}
	catBreakdown := BreakdownTable{
		Title:   "Control Results by Category",
		Columns: []string{"Category", "Pass", "Fail", "Partial", "N/A"},
	}
	for cat, ca := range catMap {
		catBreakdown.Rows = append(catBreakdown.Rows, []string{
			cat,
			fmt.Sprintf("%d", ca.pass),
			fmt.Sprintf("%d", ca.fail),
			fmt.Sprintf("%d", ca.partial),
			fmt.Sprintf("%d", ca.na),
		})
	}

	// Highlights: overdue controls.
	highlights := HighlightSection{
		Title:       "Overdue Controls",
		Description: "Controls past their SLA deadline requiring remediation",
		Columns:     []string{"Control ID", "Category", "Status", "Days Overdue", "Remediation Hint"},
	}
	for _, c := range frameworkOverdue {
		daysOverdue := ""
		if c.DaysOverdue.Valid {
			daysOverdue = fmt.Sprintf("%d", c.DaysOverdue.Int32)
		}
		highlights.Rows = append(highlights.Rows, []string{
			c.ControlID,
			c.Category,
			c.Status,
			daysOverdue,
			pgTextStr(c.RemediationHint),
		})
		highlights.RowColors = append(highlights.RowColors, ColorRed)
	}

	// Detail table: non-compliant endpoints.
	detail := DetailTable{
		Columns: []string{
			"Hostname", "OS", "Score", "Compliant CVEs", "At Risk", "Non Compliant",
		},
		MaxPDFRows: 500,
		TotalRows:  len(ncEndpoints),
	}
	for i, ep := range ncEndpoints {
		if i >= 500 {
			break
		}
		epScore := numericToFloat(ep.Score)
		detail.Rows = append(detail.Rows, []string{
			ep.Hostname,
			ep.OsFamily,
			fmt.Sprintf("%.1f%%", epScore),
			fmt.Sprintf("%d", ep.CompliantCves),
			fmt.Sprintf("%d", ep.AtRiskCves),
			fmt.Sprintf("%d", ep.NonCompliantCves),
		})
		detail.RowColors = append(detail.RowColors, ScoreColor(epScore))
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "Compliance Report",
			Subtitle:    frameworkID,
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{controlChart, trendChart},
		Breakdowns: []BreakdownTable{catBreakdown},
		Highlights: []HighlightSection{highlights},
		Detail:     detail,
	}, nil
}
