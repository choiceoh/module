package report

import (
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

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borders(),
	})
	missingStyle, _ := f.NewStyle(&excelize.Style{
		Fill:   excelize.Fill{Type: "pattern", Color: []string{"#FFF2CC"}, Pattern: 1},
		Border: borders(),
	})

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, widths[i])
	}

	result := &BuildResult{Lines: make([]Line, 0, len(serials))}
	for i, serial := range serials {
		row := i + 2
		no := ""
		if meta.AutoNumber {
			no = itoa(i + 1)
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
			cell, _ := excelize.CoordinatesToCellName(c+1, row)
			if v != nil {
				f.SetCellValue(sheet, cell, v)
			}
			if !ok {
				f.SetCellStyle(sheet, cell, cell, missingStyle)
			}
		}

		result.Lines = append(result.Lines, line)
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
