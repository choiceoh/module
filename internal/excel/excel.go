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

	lastColLetter := ""
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
		lastColLetter = colLetter
	}

	for r, row := range rows {
		// NO 컬럼은 정수로 저장해야 Excel 정렬이 1,2,3,…,10,11 순으로
		// 동작. 문자열로 넣으면 1,10,11,2,20,3 순으로 깨짐.
		var noVal any
		if row.PhotoNo > 0 {
			noVal = row.PhotoNo
		}
		values := []any{row.Filename, noVal, row.PalletSN, row.Serial, row.Suffix, row.Source, row.Notes}
		for i, v := range values {
			cell, err := excelize.CoordinatesToCellName(i+1, r+2)
			if err != nil {
				return fmt.Errorf("data cell (%d,%d): %w", i+1, r+2, err)
			}
			if v == nil {
				continue
			}
			f.SetCellValue(sheet, cell, v)
		}
	}

	// 헤더 행 고정 + 자동 필터 — 큰 시트에서 스크롤·정렬·필터 기본 UX.
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return fmt.Errorf("freeze header: %w", err)
	}
	if lastColLetter != "" {
		if err := f.AutoFilter(sheet, fmt.Sprintf("A1:%s1", lastColLetter), []excelize.AutoFilterOptions{}); err != nil {
			return fmt.Errorf("autofilter: %w", err)
		}
	}

	return f.SaveAs(path)
}
