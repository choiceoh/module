package excel

import (
	"io"

	"github.com/xuri/excelize/v2"

	"module-backend/internal/schema"
)

func WriteModules(w io.Writer, modules []schema.Module) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Modules"
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
		f.SetColWidth(sheet, colLetter, colLetter, 18)
	}

	for r, m := range modules {
		row := r + 2
		values := []string{
			m.ModelName, m.Manufacturer, m.Category,
			m.VoltageRated, m.CurrentRated, m.Interface,
			m.TempRange, m.Dimensions, m.Weight, m.Notes,
		}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}

	return f.Write(w)
}
