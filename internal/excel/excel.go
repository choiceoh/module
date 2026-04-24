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

	headerStyle, _ := f.NewStyle(&excelize.Style{
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

	for i, col := range schema.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col.Label)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colLetter, colLetter, col.Width)
	}

	for r, row := range rows {
		noVal := ""
		if row.PhotoNo > 0 {
			noVal = fmt.Sprintf("%d", row.PhotoNo)
		}
		values := []any{row.Filename, noVal, row.PalletSN, row.Serial, row.Suffix, row.Source, row.Notes}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, r+2)
			f.SetCellValue(sheet, cell, v)
		}
	}

	return f.SaveAs(path)
}
