package masterbook

import (
	"fmt"
	"log"
	"strings"

	"github.com/xuri/excelize/v2"
)

type MasterRow struct {
	PalletSN string  `json:"pallet_sn"`
	SN       string  `json:"sn"`
	VOC      float64 `json:"voc"`
	ISC      float64 `json:"isc"`
	PM       float64 `json:"pm"`
	VM       float64 `json:"vm"`
	IM       float64 `json:"im"`
	FF       float64 `json:"ff"`

	SheetName string `json:"sheet_name"`
	RowNumber int    `json:"row_number"`
}

type Book struct {
	Path   string
	Sheets []string
	Index  map[string]MasterRow
}

type LoadSummary struct {
	Path      string   `json:"path"`
	Sheets    []string `json:"sheets"`
	TotalRows int      `json:"total_rows"`
	IndexSize int      `json:"index_size"`
}

func Load(path string) (*Book, *LoadSummary, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("마스터 엑셀 열기 실패: %w", err)
	}
	defer f.Close()

	book := &Book{
		Path:   path,
		Sheets: f.GetSheetList(),
		Index:  map[string]MasterRow{},
	}
	summary := &LoadSummary{Path: path, Sheets: book.Sheets}

	for _, sheet := range book.Sheets {
		rows, err := f.Rows(sheet)
		if err != nil {
			return nil, nil, fmt.Errorf("시트 %s 읽기 실패: %w", sheet, err)
		}

		header := []string{}
		idx := columnMap{}
		lineNo := 0
		for rows.Next() {
			lineNo++
			cols, err := rows.Columns()
			if err != nil {
				return nil, nil, err
			}
			if lineNo == 1 {
				header = cols
				idx = columnIndex(header)
				if idx.sn < 0 {
					// No SN column → can't index. Skip the whole sheet
					// instead of silently producing 0 rows from it.
					log.Printf("masterbook: 시트 %q 에 SN 컬럼이 없어 스킵", sheet)
					break
				}
				continue
			}
			summary.TotalRows++

			row, ok := parseRow(cols, idx, sheet, lineNo)
			if !ok {
				continue
			}
			if existing, dup := book.Index[row.SN]; dup {
				row.RowNumber = existing.RowNumber
			}
			book.Index[row.SN] = row
		}
		if err := rows.Close(); err != nil {
			return nil, nil, err
		}
	}

	summary.IndexSize = len(book.Index)
	if summary.IndexSize == 0 {
		return nil, nil, fmt.Errorf("마스터에서 SN 컬럼을 찾지 못했거나 데이터가 비어 있습니다 (검사한 시트: %d개)", len(book.Sheets))
	}
	return book, summary, nil
}

type columnMap struct {
	pallet, sn, voc, isc, pm, vm, im, ff int
}

func columnIndex(header []string) columnMap {
	m := columnMap{pallet: -1, sn: -1, voc: -1, isc: -1, pm: -1, vm: -1, im: -1, ff: -1}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		switch key {
		case "pallet sn", "palletsn", "pallet_sn":
			m.pallet = i
		case "sn", "serial", "serial number":
			m.sn = i
		case "voc":
			m.voc = i
		case "isc":
			m.isc = i
		case "pm":
			m.pm = i
		case "vm":
			m.vm = i
		case "im":
			m.im = i
		case "ff":
			m.ff = i
		}
	}
	return m
}

func parseRow(cols []string, idx columnMap, sheet string, line int) (MasterRow, bool) {
	get := func(i int) string {
		if i < 0 || i >= len(cols) {
			return ""
		}
		return strings.TrimSpace(cols[i])
	}
	sn := get(idx.sn)
	if sn == "" {
		return MasterRow{}, false
	}
	return MasterRow{
		PalletSN:  get(idx.pallet),
		SN:        sn,
		VOC:       parseFloat(get(idx.voc)),
		ISC:       parseFloat(get(idx.isc)),
		PM:        parseFloat(get(idx.pm)),
		VM:        parseFloat(get(idx.vm)),
		IM:        parseFloat(get(idx.im)),
		FF:        parseFloat(get(idx.ff)),
		SheetName: sheet,
		RowNumber: line,
	}, true
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	if err != nil {
		return 0
	}
	return v
}

func (b *Book) Lookup(sn string) (MasterRow, bool) {
	row, ok := b.Index[strings.TrimSpace(sn)]
	return row, ok
}
