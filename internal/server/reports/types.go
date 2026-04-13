package reports

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ReportType identifies the kind of report.
type ReportType string

const (
	ReportEndpoints   ReportType = "endpoints"
	ReportPatches     ReportType = "patches"
	ReportCVEs        ReportType = "cves"
	ReportDeployments ReportType = "deployments"
	ReportCompliance  ReportType = "compliance"
	ReportExecutive   ReportType = "executive"
)

// ReportFormat identifies the output format.
type ReportFormat string

const (
	FormatPDF  ReportFormat = "pdf"
	FormatCSV  ReportFormat = "csv"
	FormatXLSX ReportFormat = "xlsx"
)

// IST is Indian Standard Time (UTC+5:30), used for all report timestamps.
var IST = time.FixedZone("IST", 5*60*60+30*60)

// ReportMeta holds header information rendered on every report.
type ReportMeta struct {
	Title       string
	Subtitle    string // e.g., "PCI DSS v4.0" for compliance reports
	TenantName  string
	DateFrom    time.Time
	DateTo      time.Time
	GeneratedAt time.Time // always in IST
	GeneratedBy string    // user name or email
}

// StatBox is a summary metric displayed in the report header area.
type StatBox struct {
	Label string
	Value string // formatted: "247", "87.3%", etc.
	Trend string // "up", "down", "flat", or "" for no trend
	Color string // "green", "red", "orange", "blue", "gray"
}

// ChartSpec describes a chart to render as a PNG and embed in the PDF.
type ChartSpec struct {
	Title  string
	Type   string // "pie", "bar", "horizontal_bar", "line", "gauge"
	Data   []ChartDataPoint
	Width  int // pixels
	Height int // pixels
}

// ChartDataPoint is a single data point in a chart.
type ChartDataPoint struct {
	Label string
	Value float64
	Color string // hex color, e.g. "#e53e3e"
}

// BreakdownTable is a grouped summary table (e.g., "By OS Family").
type BreakdownTable struct {
	Title     string
	Columns   []string
	Rows      [][]string
	RowColors []string // optional per-row background hint
}

// HighlightSection is an "attention required" block with critical items.
type HighlightSection struct {
	Title       string
	Description string
	Columns     []string
	Rows        [][]string
	RowColors   []string
}

// DetailTable is the full data table at the bottom of the report.
type DetailTable struct {
	Columns    []string
	Rows       [][]string
	RowColors  []string // per-row color hints for severity
	MaxPDFRows int      // cap for PDF rendering (default 500)
	TotalRows  int      // actual count (shown as "Showing X of Y")
}

// ReportData is the complete, format-agnostic report data passed to renderers.
type ReportData struct {
	Meta       ReportMeta
	Summary    []StatBox
	Charts     []ChartSpec
	Breakdowns []BreakdownTable
	Highlights []HighlightSection
	Detail     DetailTable
}

// Renderer converts a ReportData into a byte slice in a specific format.
type Renderer interface {
	Render(data *ReportData) ([]byte, error)
	ContentType() string
	FileExtension() string
}

// Assembler builds a ReportData from database queries for a specific report type.
type Assembler interface {
	Assemble(ctx context.Context, opts AssembleOptions) (*ReportData, error)
}

// AssembleOptions are parameters passed to an assembler.
type AssembleOptions struct {
	TenantID    string
	TenantName  string
	GeneratedBy string
	Filters     ReportFilters
}

// ReportFilters are optional filters applied when generating a report.
type ReportFilters struct {
	Status           string `json:"status,omitempty"`
	Severity         string `json:"severity,omitempty"`
	OSFamily         string `json:"os_family,omitempty"`
	DateFrom         string `json:"date_from,omitempty"` // RFC3339
	DateTo           string `json:"date_to,omitempty"`   // RFC3339
	FrameworkID      string `json:"framework_id,omitempty"`
	TagID            string `json:"tag_id,omitempty"`
	ExploitAvailable string `json:"exploit_available,omitempty"`
	CISAKev          string `json:"cisa_kev,omitempty"`
	AttackVector     string `json:"attack_vector,omitempty"`
	HasPatch         string `json:"has_patch,omitempty"`
}

// GenerateRequest is the JSON body for POST /api/v1/reports/generate.
type GenerateRequest struct {
	ReportType ReportType    `json:"report_type"`
	Format     ReportFormat  `json:"format"`
	Filters    ReportFilters `json:"filters"`
}

// GenerateResponse is the JSON response for a generation request.
type GenerateResponse struct {
	ID         string `json:"id"`
	ReportType string `json:"report_type"`
	Format     string `json:"format"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

// ReportRecord represents a row from the report_generations table.
type ReportRecord struct {
	ID             string        `json:"id"`
	TenantID       string        `json:"tenant_id"`
	ReportType     string        `json:"report_type"`
	Format         string        `json:"format"`
	Status         string        `json:"status"`
	Name           string        `json:"name"`
	Filters        ReportFilters `json:"filters"`
	FilePath       string        `json:"file_path,omitempty"`
	FileSizeBytes  int64         `json:"file_size_bytes,omitempty"`
	ChecksumSHA256 string        `json:"checksum_sha256,omitempty"`
	RowCount       int           `json:"row_count,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	CreatedBy      string        `json:"created_by"`
	CreatedAt      string        `json:"created_at"`
	CompletedAt    string        `json:"completed_at,omitempty"`
	ExpiresAt      string        `json:"expires_at"`
}

// numericToFloat converts a pgtype.Numeric to float64, returning 0 if invalid.
func numericToFloat(n pgtype.Numeric) float64 {
	f8, err := n.Float64Value()
	if err != nil || !f8.Valid {
		return 0
	}
	return f8.Float64
}

// Severity color constants used for row coloring.
const (
	ColorRed    = "#e53e3e"
	ColorOrange = "#dd6b20"
	ColorYellow = "#d69e2e"
	ColorGreen  = "#38a169"
	ColorBlue   = "#3182ce"
	ColorGray   = "#718096"
)

// SeverityColor maps a severity string to a hex color.
func SeverityColor(severity string) string {
	switch severity {
	case "critical":
		return ColorRed
	case "high":
		return ColorOrange
	case "medium":
		return ColorYellow
	case "low":
		return ColorGreen
	default:
		return ColorGray
	}
}

// ScoreColor returns a color based on a percentage score.
func ScoreColor(pct float64) string {
	switch {
	case pct < 50:
		return ColorRed
	case pct < 75:
		return ColorOrange
	case pct < 90:
		return ColorYellow
	default:
		return ColorGreen
	}
}
