package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// DeploymentsQuerier defines the database methods needed by the deployments assembler.
type DeploymentsQuerier interface {
	ListDeploymentsByTenantFiltered(ctx context.Context, arg sqlcgen.ListDeploymentsByTenantFilteredParams) ([]sqlcgen.Deployment, error)
	CountDeploymentsByTenantFiltered(ctx context.Context, arg sqlcgen.CountDeploymentsByTenantFilteredParams) (int64, error)
	CountDeploymentsByStatus(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountDeploymentsByStatusRow, error)
}

// DeploymentsAssembler builds a ReportData for the deployments report.
type DeploymentsAssembler struct {
	q DeploymentsQuerier
}

// NewDeploymentsAssembler creates a new DeploymentsAssembler.
func NewDeploymentsAssembler(q DeploymentsQuerier) *DeploymentsAssembler {
	return &DeploymentsAssembler{q: q}
}

// Assemble fetches deployment data and builds a ReportData.
func (a *DeploymentsAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble deployments report: parse tenant id: %w", err)
	}

	var createdAfter, createdBefore pgtype.Timestamptz
	if opts.Filters.DateFrom != "" {
		if t, err := time.Parse(time.RFC3339, opts.Filters.DateFrom); err == nil {
			createdAfter = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if opts.Filters.DateTo != "" {
		if t, err := time.Parse(time.RFC3339, opts.Filters.DateTo); err == nil {
			createdBefore = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	deployments, err := a.q.ListDeploymentsByTenantFiltered(ctx, sqlcgen.ListDeploymentsByTenantFilteredParams{
		TenantID:      tid,
		Status:        opts.Filters.Status,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
		PageLimit:     10000,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble deployments report: list deployments: %w", err)
	}

	totalCount, err := a.q.CountDeploymentsByTenantFiltered(ctx, sqlcgen.CountDeploymentsByTenantFilteredParams{
		TenantID:      tid,
		Status:        opts.Filters.Status,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble deployments report: count deployments: %w", err)
	}

	statusCounts, err := a.q.CountDeploymentsByStatus(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble deployments report: count by status: %w", err)
	}

	now := time.Now().In(IST)
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	// Aggregate stats.
	var succeeded, failed, pending int32
	var totalDuration time.Duration
	var durationCount int
	for _, d := range deployments {
		switch d.Status {
		case "completed", "success":
			succeeded++
		case "failed":
			failed++
		case "created", "running", "scheduled":
			pending++
		}
		if d.StartedAt.Valid && d.CompletedAt.Valid {
			totalDuration += d.CompletedAt.Time.Sub(d.StartedAt.Time)
			durationCount++
		}
	}
	successPct := float64(0)
	if len(deployments) > 0 {
		successPct = float64(succeeded) * 100 / float64(len(deployments))
	}
	avgDuration := ""
	if durationCount > 0 {
		avg := totalDuration / time.Duration(durationCount)
		avgDuration = formatDuration(avg)
	} else {
		avgDuration = "N/A"
	}

	// Subtitle.
	var subtitleParts []string
	if opts.Filters.Status != "" {
		subtitleParts = append(subtitleParts, "Status: "+opts.Filters.Status)
	}
	subtitle := strings.Join(subtitleParts, " | ")

	stats := []StatBox{
		{Label: "Total Deployments", Value: fmt.Sprintf("%d", totalCount), Color: "blue"},
		{Label: "Succeeded", Value: fmt.Sprintf("%d (%.1f%%)", succeeded, successPct), Color: "green"},
		{Label: "Failed", Value: fmt.Sprintf("%d", failed), Color: "red"},
		{Label: "Pending", Value: fmt.Sprintf("%d", pending), Color: "orange"},
		{Label: "Avg Duration", Value: avgDuration, Color: "gray"},
	}

	// Status distribution chart.
	statusChart := ChartSpec{
		Title:  "Status Distribution",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	statusColors := map[string]string{
		"completed":       ColorGreen,
		"success":         ColorGreen,
		"failed":          ColorRed,
		"running":         ColorBlue,
		"created":         ColorGray,
		"scheduled":       ColorYellow,
		"cancelled":       ColorGray,
		"rolling_back":    ColorOrange,
		"rolled_back":     ColorOrange,
		"rollback_failed": ColorRed,
	}
	for _, sc := range statusCounts {
		color := statusColors[sc.Status]
		if color == "" {
			color = ColorGray
		}
		statusChart.Data = append(statusChart.Data, ChartDataPoint{
			Label: sc.Status,
			Value: float64(sc.Count),
			Color: color,
		})
	}

	// Breakdown by status.
	statusBreakdown := BreakdownTable{
		Title:   "By Status",
		Columns: []string{"Status", "Count"},
	}
	for _, sc := range statusCounts {
		statusBreakdown.Rows = append(statusBreakdown.Rows, []string{
			sc.Status,
			fmt.Sprintf("%d", sc.Count),
		})
		color := statusColors[sc.Status]
		if color == "" {
			color = ColorGray
		}
		statusBreakdown.RowColors = append(statusBreakdown.RowColors, color)
	}

	// Highlights: failed deployments.
	highlights := HighlightSection{
		Title:       "Failed Deployments",
		Description: "Deployments that failed and may require investigation",
		Columns:     []string{"Name", "Total Targets", "Failed", "Started", "Completed"},
	}
	for _, d := range deployments {
		if d.Status != "failed" {
			continue
		}
		name := pgTextStr(d.Name)
		if name == "" {
			name = fmt.Sprintf("Deployment %s", uuidStr(d.ID))
		}
		started := ""
		if d.StartedAt.Valid {
			started = d.StartedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}
		completed := ""
		if d.CompletedAt.Valid {
			completed = d.CompletedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}
		highlights.Rows = append(highlights.Rows, []string{
			name,
			fmt.Sprintf("%d", d.TotalTargets),
			fmt.Sprintf("%d", d.FailedCount),
			started,
			completed,
		})
		highlights.RowColors = append(highlights.RowColors, ColorRed)
	}

	// Detail table.
	detail := DetailTable{
		Columns: []string{
			"Name", "Status", "Total Targets", "Succeeded", "Failed",
			"Pending", "Started", "Completed", "Duration",
		},
		MaxPDFRows: 500,
		TotalRows:  int(totalCount),
	}
	for i, d := range deployments {
		if i >= 500 {
			break
		}
		name := pgTextStr(d.Name)
		if name == "" {
			name = fmt.Sprintf("Deployment %s", uuidStr(d.ID))
		}
		started := ""
		if d.StartedAt.Valid {
			started = d.StartedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}
		completed := ""
		if d.CompletedAt.Valid {
			completed = d.CompletedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}
		duration := ""
		if d.StartedAt.Valid && d.CompletedAt.Valid {
			duration = formatDuration(d.CompletedAt.Time.Sub(d.StartedAt.Time))
		}
		pendingCount := d.TotalTargets - d.SuccessCount - d.FailedCount
		if pendingCount < 0 {
			pendingCount = 0
		}

		detail.Rows = append(detail.Rows, []string{
			name,
			d.Status,
			fmt.Sprintf("%d", d.TotalTargets),
			fmt.Sprintf("%d", d.SuccessCount),
			fmt.Sprintf("%d", d.FailedCount),
			fmt.Sprintf("%d", pendingCount),
			started,
			completed,
			duration,
		})

		switch d.Status {
		case "failed", "rollback_failed":
			detail.RowColors = append(detail.RowColors, ColorRed)
		case "rolling_back", "rolled_back":
			detail.RowColors = append(detail.RowColors, ColorOrange)
		case "completed", "success":
			detail.RowColors = append(detail.RowColors, ColorGreen)
		default:
			detail.RowColors = append(detail.RowColors, ColorGray)
		}
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "Deployments Report",
			Subtitle:    subtitle,
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{statusChart},
		Breakdowns: []BreakdownTable{statusBreakdown},
		Highlights: []HighlightSection{highlights},
		Detail:     detail,
	}, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func uuidStr(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
