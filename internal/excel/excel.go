package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"module-scanner/internal/schema"
)

func WriteScanResults(path string, rows []schema.ScanResult) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Scans"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#BFBFBF", Style: 1},
			{Type: "right", Color: "#BFBFBF", Style: 1},
			{Type: "top", Color: "#BFBFBF", Style: 1},
			{Type: "bottom", Color: "#BFBFBF", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("create header style: %w", err)
	}

	for i, col := range schema.Columns {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return fmt.Errorf("header cell %d: %w", i+1, err)
		}
		f.SetCellValue(sheet, cell, col.Label)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		colLetter, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return fmt.Errorf("header col name %d: %w", i+1, err)
		}
		f.SetColWidth(sheet, colLetter, colLetter, col.Width)
	}

	for r, row := range rows {
		noVal := ""
		if row.PhotoNo > 0 {
			noVal = fmt.Sprintf("%d", row.PhotoNo)
		}
		values := []any{row.Filename, noVal, row.PalletSN, row.Serial, row.Suffix, row.Source, row.Notes}
		for i, v := range values {
			cell, err := excelize.CoordinatesToCellName(i+1, r+2)
			if err != nil {
				return fmt.Errorf("data cell (%d,%d): %w", i+1, r+2, err)
			}
			f.SetCellValue(sheet, cell, v)
		}
	}

	return f.SaveAs(path)
}
