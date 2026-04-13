package reports

import (
	"bytes"
	"encoding/csv"
	"fmt"
)

// CSVRenderer renders a ReportData as CSV, outputting only the detail table.
type CSVRenderer struct{}

// Render produces a CSV byte slice from the detail table in data.
func (r *CSVRenderer) Render(data *ReportData) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	// Comment row with report metadata.
	comment := fmt.Sprintf("# Report: %s | Generated: %s | Tenant: %s",
		data.Meta.Title,
		data.Meta.GeneratedAt.In(IST).Format("2006-01-02 15:04:05 MST"),
		data.Meta.TenantName,
	)
	if err := w.Write([]string{comment}); err != nil {
		return nil, fmt.Errorf("csv render: write comment: %w", err)
	}

	// Header row.
	if err := w.Write(data.Detail.Columns); err != nil {
		return nil, fmt.Errorf("csv render: write headers: %w", err)
	}

	// Data rows (no cap — CSV gets everything).
	for i, row := range data.Detail.Rows {
		if err := w.Write(row); err != nil {
			return nil, fmt.Errorf("csv render: write row %d: %w", i, err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("csv render: flush: %w", err)
	}

	return buf.Bytes(), nil
}

// ContentType returns the MIME type for CSV.
func (r *CSVRenderer) ContentType() string {
	return "text/csv; charset=utf-8"
}

// FileExtension returns "csv".
func (r *CSVRenderer) FileExtension() string {
	return "csv"
}
