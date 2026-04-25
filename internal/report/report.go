package report

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"module-scanner/internal/masterbook"
)

type Meta struct {
	ModuleType  string `json:"module_type"`
	CellType    string `json:"cell_type"`
	ProjectName string `json:"project_name"`
	AutoNumber  bool   `json:"auto_number"`
}

type Line struct {
	Serial  string `json:"serial"`
	Found   bool   `json:"found"`
	Warning string `json:"warning,omitempty"`
}

type BuildResult struct {
	Path    string `json:"path"`
	Matched int    `json:"matched"`
	Missing int    `json:"missing"`
	Lines   []Line `json:"lines"`
}

var headers = []string{
	"NO", "Pallet Sn", "Module type", "Cell Type", "SN",
	"VOC", "ISC", "PM", "VM", "IM", "FF", "프로젝트명",
}

var widths = []float64{6, 16, 22, 14, 28, 10, 10, 10, 10, 10, 8, 14}

func Build(path string, serials []string, book *masterbook.Book, meta Meta) (*BuildResult, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"
	f.SetSheetName(f.GetSheetName(0), sheet)

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create header style: %w", err)
	}
	missingStyle, err := f.NewStyle(&excelize.Style{
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"#FFF2CC"}, Pattern: 1},
		Border: borders(),
	})
	if err != nil {
		return nil, fmt.Errorf("create missing-row style: %w", err)
	}

	lastColLetter := ""
	for i, h := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return nil, fmt.Errorf("header cell %d: %w", i+1, err)
		}
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return nil, fmt.Errorf("header col name %d: %w", i+1, err)
		}
		f.SetColWidth(sheet, col, col, widths[i])
		lastColLetter = col
	}

	result := &BuildResult{Lines: make([]Line, 0, len(serials))}
	for i, serial := range serials {
		row := i + 2
		// NO 컬럼은 정수로 저장해야 Excel 정렬이 1,2,3,…,10,11 순으로
		// 동작. 문자열로 넣으면 1,10,11,2,20,3 순으로 깨짐.
		var no any
		if meta.AutoNumber {
			no = i + 1
		}

		master, ok := book.Lookup(serial)
		line := Line{Serial: serial, Found: ok}

		values := make([]any, 12)
		values[0] = no
		values[2] = meta.ModuleType
		values[3] = meta.CellType
		values[4] = serial
		values[11] = meta.ProjectName

		if ok {
			values[1] = master.PalletSN
			values[5] = master.VOC
			values[6] = master.ISC
			values[7] = master.PM
			values[8] = master.VM
			values[9] = master.IM
			values[10] = master.FF
			result.Matched++
		} else {
			line.Warning = "마스터에 없음"
			result.Missing++
		}

		for c, v := range values {
			cell, err := excelize.CoordinatesToCellName(c+1, row)
			if err != nil {
				return nil, fmt.Errorf("data cell (%d,%d): %w", c+1, row, err)
			}
			if v != nil {
				f.SetCellValue(sheet, cell, v)
			}
			if !ok {
				f.SetCellStyle(sheet, cell, cell, missingStyle)
			}
		}

		result.Lines = append(result.Lines, line)
	}

	// 헤더 행 고정 + 자동 필터 — 큰 시트에서 스크롤·정렬·필터 기본 UX.
	if err := f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, fmt.Errorf("freeze header: %w", err)
	}
	if lastColLetter != "" {
		if err := f.AutoFilter(sheet, fmt.Sprintf("A1:%s1", lastColLetter), []excelize.AutoFilterOptions{}); err != nil {
			return nil, fmt.Errorf("autofilter: %w", err)
		}
	}

	if err := f.SaveAs(path); err != nil {
		return nil, err
	}
	result.Path = path
	return result, nil
}

func borders() []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: "#BFBFBF", Style: 1},
		{Type: "right", Color: "#BFBFBF", Style: 1},
		{Type: "top", Color: "#BFBFBF", Style: 1},
		{Type: "bottom", Color: "#BFBFBF", Style: 1},
	}
}
