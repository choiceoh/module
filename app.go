package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	rt "github.com/wailsapp/wails/v2/pkg/runtime"

	"module-scanner/internal/barcode"
	"module-scanner/internal/excel"
	"module-scanner/internal/masterbook"
	"module-scanner/internal/ocr"
	"module-scanner/internal/presets"
	"module-scanner/internal/report"
	"module-scanner/internal/schema"
)

type App struct {
	ctx context.Context

	mu      sync.Mutex
	master  *masterbook.Book
	presets *presets.Store
}

func NewApp() *App {
	store, _ := presets.Load()
	return &App{presets: store}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Version() string {
	return appVersion()
}

func (a *App) ScanImage(filename, dataURL string) []schema.ScanResult {
	data, err := decodeDataURL(dataURL)
	if err != nil {
		return []schema.ScanResult{{Filename: filename, Error: "invalid data url: " + err.Error()}}
	}

	// OCR-only: run PaddleOCR sidecar (bundled rapidocr_onnxruntime CLI).
	results, err := ocr.RecognizeBytes(data)
	if err != nil {
		return []schema.ScanResult{{Filename: filename, Error: "ocr: " + err.Error()}}
	}

	// Pair each serial-looking result with the nearest short-digit
	// result (NO-column). Standard packing list layout: NO on the left
	// of SN on the same row, so we look for digit-only text immediately
	// to the left of each serial.
	noBoxesByText := map[string]ocr.Result{}
	for _, r := range results {
		t := strings.TrimSpace(r.Text)
		if isNoNumber(t) {
			noBoxesByText[t] = r
		}
	}
	var noResults []ocr.Result
	for _, r := range noBoxesByText {
		noResults = append(noResults, r)
	}

	// Extract alphanumeric serial candidates
	sourceBy := map[string]string{}
	noBy := map[string]int{}
	correctedBy := map[string]string{}
	boxBy := map[string]ocr.Result{}
	for _, r := range results {
		for _, s := range extractAlnumRuns(strings.ToUpper(strings.TrimSpace(r.Text)), 6, 40) {
			if _, has := sourceBy[s]; has {
				continue
			}
			sourceBy[s] = "ocr"
			boxBy[s] = r
		}
	}
	// Assign NO by spatial proximity: pick the digit-only box whose Y
	// range overlaps with the serial's and whose X is to the left.
	for s, sr := range boxBy {
		cy := (sr.Y0 + sr.Y1) / 2
		bestN := 0
		bestDx := 1 << 30
		for _, nb := range noResults {
			ny := (nb.Y0 + nb.Y1) / 2
			if abs(ny-cy) > (sr.Y1-sr.Y0)/2+(nb.Y1-nb.Y0)/2+8 {
				continue
			}
			if nb.X1 > sr.X0 {
				continue // not to the left
			}
			dx := sr.X0 - nb.X1
			if dx < bestDx {
				bestDx = dx
				n, _ := atoiSafe(strings.TrimSpace(nb.Text))
				bestN = n
			}
		}
		if bestN > 0 {
			noBy[s] = bestN
		}
	}

	if len(sourceBy) == 0 {
		return []schema.ScanResult{{Filename: filename, Error: "no serials detected"}}
	}

	allTexts := make([]string, 0, len(sourceBy))
	for t := range sourceBy {
		allTexts = append(allTexts, t)
	}
	modules, pallet, _ := barcode.Classify(allTexts)

	if len(modules) == 0 {
		if pallet != "" {
			return []schema.ScanResult{{
				Filename: filename,
				PalletSN: pallet,
				Source:   "barcode",
				Notes:    "팔레트 번호만 검출 (모듈 시리얼 없음)",
			}}
		}
		return []schema.ScanResult{{Filename: filename, Error: "no module serials detected"}}
	}

	// Sort modules by VLM-provided NO when available; unknowns go to the end.
	sort.SliceStable(modules, func(i, j int) bool {
		ni, okI := noBy[modules[i]]
		nj, okJ := noBy[modules[j]]
		switch {
		case okI && okJ:
			return ni < nj
		case okI:
			return true
		case okJ:
			return false
		default:
			return modules[i] < modules[j]
		}
	})

	out := make([]schema.ScanResult, 0, len(modules))
	for _, s := range modules {
		serial, suffix := splitSuffix(s)
		src := sourceBy[s]
		if src == "" {
			src = "barcode"
		}
		photoNo := 0
		if n, ok := noBy[s]; ok {
			photoNo = n
		}
		corrected := false
		rawText := ""
		if raw, ok := correctedBy[s]; ok {
			corrected = true
			rawText = raw
		}
		notes := ""
		if corrected {
			notes = fmt.Sprintf("수정됨 (원본: %s)", rawText)
		}
		score := float32(0)
		if b, ok := boxBy[s]; ok {
			score = b.Score
		}
		out = append(out, schema.ScanResult{
			Filename:  filename,
			PhotoNo:   photoNo,
			Serial:    serial,
			Suffix:    suffix,
			Source:    src,
			Score:     score,
			PalletSN:  pallet,
			Corrected: corrected,
			RawText:   rawText,
			Notes:     notes,
		})
	}
	return out
}

// isNoNumber matches text that looks like a NO-column entry (1-3 digits).
func isNoNumber(s string) bool {
	if len(s) == 0 || len(s) > 3 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// extractAlnumRuns returns uppercase alphanumeric substrings of
// reasonable length from the input text.
func extractAlnumRuns(s string, minLen, maxLen int) []string {
	var out []string
	run := make([]byte, 0, 32)
	flush := func() {
		if len(run) >= minLen && len(run) <= maxLen {
			out = append(out, string(run))
		}
		run = run[:0]
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			run = append(run, c)
		} else {
			flush()
		}
	}
	flush()
	return out
}

func atoiSafe(s string) (int, bool) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// longestCommonPrefix returns the longest prefix shared by at least
// minCount strings in the slice. Walks up one character at a time; at
// each length it picks the most-common prefix of that length and checks
// the count threshold. Used to reject OCR outputs that don't match the
// pattern validated by barcode's checksum.
func longestCommonPrefix(texts []string, minCount int) string {
	if len(texts) == 0 || minCount < 1 {
		return ""
	}
	minLen := len(texts[0])
	for _, t := range texts {
		if len(t) < minLen {
			minLen = len(t)
		}
	}
	best := ""
	for l := 1; l <= minLen; l++ {
		counts := map[string]int{}
		for _, t := range texts {
			counts[t[:l]]++
		}
		var topPrefix string
		topCount := 0
		for p, c := range counts {
			if c > topCount {
				topCount = c
				topPrefix = p
			}
		}
		if topCount >= minCount {
			best = topPrefix
		} else {
			break
		}
	}
	return best
}

func detectMIME(data []byte) string {
	if len(data) >= 8 {
		switch {
		case string(data[:3]) == "\xff\xd8\xff":
			return "image/jpeg"
		case string(data[:8]) == "\x89PNG\r\n\x1a\n":
			return "image/png"
		}
	}
	return http.DetectContentType(data)
}

func (a *App) ExportExcel(rows []schema.ScanResult) (string, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("행이 비어 있습니다")
	}

	path, err := rt.SaveFileDialog(a.ctx, rt.SaveDialogOptions{
		DefaultFilename: "scans.xlsx",
		Title:           "엑셀로 저장",
		Filters: []rt.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil
	}
	if filepath.Ext(path) == "" {
		path += ".xlsx"
	}

	if err := excel.WriteScanResults(path, rows); err != nil {
		return "", fmt.Errorf("엑셀 저장 실패: %w", err)
	}
	return path, nil
}

type MasterInfo struct {
	Path          string                 `json:"path"`
	Filename      string                 `json:"filename"`
	Sheets        []string               `json:"sheets"`
	TotalRows     int                    `json:"total_rows"`
	IndexSize     int                    `json:"index_size"`
	Preset        presets.MasterPreset   `json:"preset"`
	HasPreset     bool                   `json:"has_preset"`
	Recent        []string               `json:"recent_projects"`
	RecentPresets []presets.MasterPreset `json:"recent_presets"`
}

func (a *App) PickAndLoadMaster() (*MasterInfo, error) {
	path, err := rt.OpenFileDialog(a.ctx, rt.OpenDialogOptions{
		Title: "마스터 엑셀 선택",
		Filters: []rt.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	return a.LoadMaster(path)
}

func (a *App) LoadMaster(path string) (*MasterInfo, error) {
	book, summary, err := masterbook.Load(path)
	if err != nil {
		return nil, fmt.Errorf("마스터 로드 실패: %w", err)
	}

	a.mu.Lock()
	a.master = book
	a.mu.Unlock()

	info := &MasterInfo{
		Path:          summary.Path,
		Filename:      filepath.Base(summary.Path),
		Sheets:        summary.Sheets,
		TotalRows:     summary.TotalRows,
		IndexSize:     summary.IndexSize,
		Recent:        a.presets.RecentProjects,
		RecentPresets: a.presets.RecentPresets,
	}
	if info.Recent == nil {
		info.Recent = []string{}
	}
	if info.RecentPresets == nil {
		info.RecentPresets = []presets.MasterPreset{}
	}
	if p, ok := a.presets.GetMaster(path); ok {
		info.Preset = p
		info.HasPreset = true
	}
	return info, nil
}

type BuildRequest struct {
	Serials     []string `json:"serials"`
	ModuleType  string   `json:"module_type"`
	CellType    string   `json:"cell_type"`
	ProjectName string   `json:"project_name"`
	AutoNumber  bool     `json:"auto_number"`
	SavePreset  bool     `json:"save_preset"`
}

func (a *App) BuildReport(req BuildRequest) (*report.BuildResult, error) {
	a.mu.Lock()
	book := a.master
	a.mu.Unlock()

	if book == nil {
		return nil, fmt.Errorf("먼저 마스터 엑셀을 불러오세요")
	}
	if len(req.Serials) == 0 {
		return nil, fmt.Errorf("시리얼이 비어 있습니다")
	}

	defaultName := "report.xlsx"
	if req.ProjectName != "" {
		defaultName = fmt.Sprintf("%s %d매.xlsx", req.ProjectName, len(req.Serials))
	}

	path, err := rt.SaveFileDialog(a.ctx, rt.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "리포트 엑셀 저장",
		Filters: []rt.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	if filepath.Ext(path) == "" {
		path += ".xlsx"
	}

	res, err := report.Build(path, req.Serials, book, report.Meta{
		ModuleType:  req.ModuleType,
		CellType:    req.CellType,
		ProjectName: req.ProjectName,
		AutoNumber:  req.AutoNumber,
	})
	if err != nil {
		return nil, err
	}

	preset := presets.MasterPreset{
		ModuleType: req.ModuleType,
		CellType:   req.CellType,
	}
	if req.SavePreset && book.Path != "" {
		a.presets.PutMaster(book.Path, preset)
	}
	a.presets.PushRecentProject(req.ProjectName)
	a.presets.PushRecentPreset(preset)
	_ = a.presets.Save()

	return res, nil
}

func decodeDataURL(s string) ([]byte, error) {
	i := strings.Index(s, ",")
	if i < 0 {
		return nil, fmt.Errorf("not a data url")
	}
	return base64.StdEncoding.DecodeString(s[i+1:])
}

func splitSuffix(text string) (string, string) {
	text = strings.TrimSpace(text)
	for _, sep := range []string{"|", " ", "\t"} {
		if i := strings.LastIndex(text, sep); i >= 0 && len(text)-i-1 <= 2 && len(text)-i-1 >= 1 {
			return strings.TrimSpace(text[:i]), strings.TrimSpace(text[i+1:])
		}
	}
	return text, ""
}
