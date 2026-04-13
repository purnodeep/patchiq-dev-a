package reports

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// XLSXRenderer renders a ReportData as an Excel workbook.
type XLSXRenderer struct{}

// colorMap maps StatBox color names to hex fill colors.
var colorMap = map[string]string{
	"green":  "#38a169",
	"red":    "#e53e3e",
	"orange": "#dd6b20",
	"blue":   "#3182ce",
	"gray":   "#718096",
}

// Render produces an XLSX byte slice from data.
func (r *XLSXRenderer) Render(data *ReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: Summary (default sheet renamed).
	summarySheet := "Summary"
	if idx, err := f.GetSheetIndex("Sheet1"); err == nil && idx >= 0 {
		if err := f.SetSheetName("Sheet1", summarySheet); err != nil {
			return nil, fmt.Errorf("xlsx render: rename sheet: %w", err)
		}
	}

	// -- Title row (merged across 4 columns, bold 16pt) --
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16},
	})
	if err != nil {
		return nil, fmt.Errorf("xlsx render: title style: %w", err)
	}
	if err := f.MergeCell(summarySheet, "A1", "D1"); err != nil {
		return nil, fmt.Errorf("xlsx render: merge title: %w", err)
	}
	_ = f.SetCellValue(summarySheet, "A1", data.Meta.Title)
	_ = f.SetCellStyle(summarySheet, "A1", "D1", titleStyle)

	// -- Subtitle/meta row --
	meta := fmt.Sprintf("Generated: %s | Tenant: %s | By: %s",
		data.Meta.GeneratedAt.In(IST).Format("2006-01-02 15:04:05 MST"),
		data.Meta.TenantName,
		data.Meta.GeneratedBy,
	)
	_ = f.SetCellValue(summarySheet, "A2", meta)

	// -- Summary stat boxes (row 4 = labels, row 5 = values) --
	for i, stat := range data.Summary {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		labelCell := colName + "4"
		valueCell := colName + "5"
		_ = f.SetCellValue(summarySheet, labelCell, stat.Label)
		_ = f.SetCellValue(summarySheet, valueCell, stat.Value)

		// Apply fill color.
		if hex, ok := colorMap[stat.Color]; ok {
			style, sErr := f.NewStyle(&excelize.Style{
				Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{hex}},
				Font: &excelize.Font{Bold: true, Color: "#FFFFFF"},
			})
			if sErr == nil {
				_ = f.SetCellStyle(summarySheet, valueCell, valueCell, style)
			}
		}
	}

	// -- Breakdown tables starting at row 7 --
	grayHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#f7fafc"}},
		Font: &excelize.Font{Bold: true},
	})
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
	})

	curRow := 7
	for _, bt := range data.Breakdowns {
		// Title row (merged).
		startCol, _ := excelize.ColumnNumberToName(1)
		endCol, _ := excelize.ColumnNumberToName(max(len(bt.Columns), 1))
		titleCell := fmt.Sprintf("%s%d", startCol, curRow)
		endCell := fmt.Sprintf("%s%d", endCol, curRow)
		_ = f.MergeCell(summarySheet, titleCell, endCell)
		_ = f.SetCellValue(summarySheet, titleCell, bt.Title)
		_ = f.SetCellStyle(summarySheet, titleCell, endCell, boldStyle)
		curRow++

		// Header row.
		for ci, col := range bt.Columns {
			cn, _ := excelize.ColumnNumberToName(ci + 1)
			cell := fmt.Sprintf("%s%d", cn, curRow)
			_ = f.SetCellValue(summarySheet, cell, col)
			_ = f.SetCellStyle(summarySheet, cell, cell, grayHeaderStyle)
		}
		curRow++

		// Data rows.
		for _, dataRow := range bt.Rows {
			for ci, val := range dataRow {
				cn, _ := excelize.ColumnNumberToName(ci + 1)
				cell := fmt.Sprintf("%s%d", cn, curRow)
				_ = f.SetCellValue(summarySheet, cell, val)
			}
			curRow++
		}
		curRow++ // blank row between tables
	}

	// Sheet 2: Data
	dataSheet := "Data"
	if _, err := f.NewSheet(dataSheet); err != nil {
		return nil, fmt.Errorf("xlsx render: new data sheet: %w", err)
	}

	// Header row.
	for ci, col := range data.Detail.Columns {
		cn, _ := excelize.ColumnNumberToName(ci + 1)
		cell := fmt.Sprintf("%s%d", cn, 1)
		_ = f.SetCellValue(dataSheet, cell, col)
		_ = f.SetCellStyle(dataSheet, cell, cell, grayHeaderStyle)
	}

	// Auto-filter on header.
	if len(data.Detail.Columns) > 0 {
		lastCol, _ := excelize.ColumnNumberToName(len(data.Detail.Columns))
		lastRow := len(data.Detail.Rows) + 1
		if lastRow < 2 {
			lastRow = 2
		}
		_ = f.AutoFilter(dataSheet, fmt.Sprintf("A1:%s%d", lastCol, lastRow), nil)
	}

	// Freeze pane on row 1.
	_ = f.SetPanes(dataSheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Data rows.
	colWidths := make([]int, len(data.Detail.Columns))
	for ci, col := range data.Detail.Columns {
		if len(col) > colWidths[ci] {
			colWidths[ci] = len(col)
		}
	}

	for ri, dataRow := range data.Detail.Rows {
		rowNum := ri + 2
		for ci, val := range dataRow {
			cn, _ := excelize.ColumnNumberToName(ci + 1)
			cell := fmt.Sprintf("%s%d", cn, rowNum)
			_ = f.SetCellValue(dataSheet, cell, val)
			if len(val) > colWidths[ci] {
				colWidths[ci] = len(val)
			}
		}

		// Apply row color if provided.
		if ri < len(data.Detail.RowColors) && data.Detail.RowColors[ri] != "" {
			rowStyle, sErr := f.NewStyle(&excelize.Style{
				Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{data.Detail.RowColors[ri]}},
			})
			if sErr == nil {
				for ci := range dataRow {
					cn, _ := excelize.ColumnNumberToName(ci + 1)
					cell := fmt.Sprintf("%s%d", cn, rowNum)
					_ = f.SetCellStyle(dataSheet, cell, cell, rowStyle)
				}
			}
		}
	}

	// Auto-width columns (cap at 40).
	for ci, w := range colWidths {
		cn, _ := excelize.ColumnNumberToName(ci + 1)
		width := float64(w) + 2
		if width > 40 {
			width = 40
		}
		_ = f.SetColWidth(dataSheet, cn, cn, width)
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("xlsx render: write buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// ContentType returns the MIME type for XLSX.
func (r *XLSXRenderer) ContentType() string {
	return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
}

// FileExtension returns "xlsx".
func (r *XLSXRenderer) FileExtension() string {
	return "xlsx"
}
