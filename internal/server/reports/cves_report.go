package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/skenzeriq/patchiq/internal/server/store/sqlcgen"
)

// CVEsQuerier defines the database methods needed by the CVEs assembler.
type CVEsQuerier interface {
	ListCVEsFiltered(ctx context.Context, arg sqlcgen.ListCVEsFilteredParams) ([]sqlcgen.ListCVEsFilteredRow, error)
	CountCVEsFiltered(ctx context.Context, arg sqlcgen.CountCVEsFilteredParams) (int64, error)
	CountCVEsBySeverity(ctx context.Context, tenantID pgtype.UUID) ([]sqlcgen.CountCVEsBySeverityRow, error)
	CountCVEsKEV(ctx context.Context, tenantID pgtype.UUID) (int32, error)
	CountCVEsExploit(ctx context.Context, tenantID pgtype.UUID) (int32, error)
}

// CVEsAssembler builds a ReportData for the CVEs report.
type CVEsAssembler struct {
	q CVEsQuerier
}

// NewCVEsAssembler creates a new CVEsAssembler.
func NewCVEsAssembler(q CVEsQuerier) *CVEsAssembler {
	return &CVEsAssembler{q: q}
}

// Assemble fetches CVE data and builds a ReportData.
func (a *CVEsAssembler) Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error) {
	tid := pgtype.UUID{}
	if err := tid.Scan(opts.TenantID); err != nil {
		return nil, fmt.Errorf("assemble cves report: parse tenant id: %w", err)
	}

	cves, err := a.q.ListCVEsFiltered(ctx, sqlcgen.ListCVEsFilteredParams{
		TenantID:         tid,
		Severity:         opts.Filters.Severity,
		CisaKev:          opts.Filters.CISAKev,
		ExploitAvailable: opts.Filters.ExploitAvailable,
		AttackVector:     opts.Filters.AttackVector,
		HasPatch:         opts.Filters.HasPatch,
		PageLimit:        10000,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble cves report: list cves: %w", err)
	}

	totalCount, err := a.q.CountCVEsFiltered(ctx, sqlcgen.CountCVEsFilteredParams{
		TenantID:         tid,
		Severity:         opts.Filters.Severity,
		CisaKev:          opts.Filters.CISAKev,
		ExploitAvailable: opts.Filters.ExploitAvailable,
		AttackVector:     opts.Filters.AttackVector,
		HasPatch:         opts.Filters.HasPatch,
	})
	if err != nil {
		return nil, fmt.Errorf("assemble cves report: count cves: %w", err)
	}

	sevCounts, err := a.q.CountCVEsBySeverity(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble cves report: count by severity: %w", err)
	}

	kevCount, err := a.q.CountCVEsKEV(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble cves report: count kev: %w", err)
	}

	exploitCount, err := a.q.CountCVEsExploit(ctx, tid)
	if err != nil {
		return nil, fmt.Errorf("assemble cves report: count exploit: %w", err)
	}

	now := time.Now().In(IST)
	dateFrom, dateTo := parseDateFilters(opts.Filters.DateFrom, opts.Filters.DateTo)

	// Count critical and no-patch.
	var criticalCount int32
	var noPatchCount int
	for _, sc := range sevCounts {
		if sc.Severity == "critical" {
			criticalCount = sc.Count
		}
	}
	for _, c := range cves {
		if !c.PatchAvailable {
			noPatchCount++
		}
	}

	// Subtitle.
	var subtitleParts []string
	if opts.Filters.Severity != "" {
		subtitleParts = append(subtitleParts, "Severity: "+opts.Filters.Severity)
	}
	if opts.Filters.AttackVector != "" {
		subtitleParts = append(subtitleParts, "Vector: "+opts.Filters.AttackVector)
	}
	subtitle := strings.Join(subtitleParts, " | ")

	stats := []StatBox{
		{Label: "Total CVEs", Value: fmt.Sprintf("%d", totalCount), Color: "blue"},
		{Label: "Critical", Value: fmt.Sprintf("%d", criticalCount), Color: "red"},
		{Label: "CISA KEV", Value: fmt.Sprintf("%d", kevCount), Color: "orange"},
		{Label: "Exploit Available", Value: fmt.Sprintf("%d", exploitCount), Color: "red"},
		{Label: "No Patch Available", Value: fmt.Sprintf("%d", noPatchCount), Color: "gray"},
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

	// Attack vector breakdown chart.
	vectorMap := make(map[string]int)
	for _, c := range cves {
		v := pgTextStr(c.AttackVector)
		if v == "" {
			v = "unknown"
		}
		vectorMap[v]++
	}
	vectorChart := ChartSpec{
		Title:  "Attack Vector Breakdown",
		Type:   "pie",
		Width:  400,
		Height: 300,
	}
	vectorColors := map[string]string{
		"NETWORK":          ColorRed,
		"ADJACENT_NETWORK": ColorOrange,
		"LOCAL":            ColorYellow,
		"PHYSICAL":         ColorGreen,
		"unknown":          ColorGray,
	}
	for v, count := range vectorMap {
		color := vectorColors[v]
		if color == "" {
			color = ColorGray
		}
		vectorChart.Data = append(vectorChart.Data, ChartDataPoint{
			Label: v,
			Value: float64(count),
			Color: color,
		})
	}

	// Breakdown by severity.
	sevBreakdown := BreakdownTable{
		Title:   "By Severity",
		Columns: []string{"Severity", "Count"},
	}
	for _, sc := range sevCounts {
		sevBreakdown.Rows = append(sevBreakdown.Rows, []string{
			sc.Severity,
			fmt.Sprintf("%d", sc.Count),
		})
		sevBreakdown.RowColors = append(sevBreakdown.RowColors, SeverityColor(sc.Severity))
	}

	// Breakdown by attack vector.
	vectorBreakdown := BreakdownTable{
		Title:   "By Attack Vector",
		Columns: []string{"Attack Vector", "Count"},
	}
	for v, count := range vectorMap {
		vectorBreakdown.Rows = append(vectorBreakdown.Rows, []string{
			v,
			fmt.Sprintf("%d", count),
		})
	}

	// Highlights: CISA KEV CVEs + CVEs with exploits.
	highlights := HighlightSection{
		Title:       "CISA KEV & Exploit Available",
		Description: "CVEs listed in CISA KEV or with known exploits, sorted by CVSS score",
		Columns:     []string{"CVE ID", "CVSS", "Severity", "KEV Due Date", "Exploit", "Affected Endpoints"},
	}
	for _, c := range cves {
		isKEV := c.CisaKevDueDate.Valid
		if !isKEV && !c.ExploitAvailable {
			continue
		}
		kevDate := ""
		if c.CisaKevDueDate.Valid {
			kevDate = c.CisaKevDueDate.Time.Format("02 Jan 2006")
		}
		exploitStr := "No"
		if c.ExploitAvailable {
			exploitStr = "Yes"
		}
		highlights.Rows = append(highlights.Rows, []string{
			c.CveID,
			numericToStr(c.CvssV3Score),
			c.Severity,
			kevDate,
			exploitStr,
			fmt.Sprintf("%d", c.AffectedEndpointCount),
		})
		highlights.RowColors = append(highlights.RowColors, SeverityColor(c.Severity))
	}

	// Detail table.
	detail := DetailTable{
		Columns: []string{
			"CVE ID", "CVSS", "Severity", "Attack Vector", "Exploit",
			"KEV Due Date", "Patch Available", "Patch Count",
			"Affected Endpoints", "Published",
		},
		MaxPDFRows: 500,
		TotalRows:  int(totalCount),
	}
	for i, c := range cves {
		if i >= 500 {
			break
		}
		kevDate := ""
		if c.CisaKevDueDate.Valid {
			kevDate = c.CisaKevDueDate.Time.Format("02 Jan 2006")
		}
		exploitStr := "No"
		if c.ExploitAvailable {
			exploitStr = "Yes"
		}
		patchAvail := "No"
		if c.PatchAvailable {
			patchAvail = "Yes"
		}
		published := ""
		if c.PublishedAt.Valid {
			published = c.PublishedAt.Time.In(IST).Format("02 Jan 2006 15:04 IST")
		}

		cvss := numericToFloat(c.CvssV3Score)

		detail.Rows = append(detail.Rows, []string{
			c.CveID,
			numericToStr(c.CvssV3Score),
			c.Severity,
			pgTextStr(c.AttackVector),
			exploitStr,
			kevDate,
			patchAvail,
			fmt.Sprintf("%d", c.PatchCount),
			fmt.Sprintf("%d", c.AffectedEndpointCount),
			published,
		})

		switch {
		case cvss >= 9:
			detail.RowColors = append(detail.RowColors, ColorRed)
		case cvss >= 7:
			detail.RowColors = append(detail.RowColors, ColorOrange)
		case cvss >= 4:
			detail.RowColors = append(detail.RowColors, ColorYellow)
		default:
			detail.RowColors = append(detail.RowColors, ColorGreen)
		}
	}

	return &ReportData{
		Meta: ReportMeta{
			Title:       "CVEs Report",
			Subtitle:    subtitle,
			TenantName:  opts.TenantName,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			GeneratedAt: now,
			GeneratedBy: opts.GeneratedBy,
		},
		Summary:    stats,
		Charts:     []ChartSpec{sevChart, vectorChart},
		Breakdowns: []BreakdownTable{sevBreakdown, vectorBreakdown},
		Highlights: []HighlightSection{highlights},
		Detail:     detail,
	}, nil
}

func numericToStr(n pgtype.Numeric) string {
	f := numericToFloat(n)
	return fmt.Sprintf("%.1f", f)
}
