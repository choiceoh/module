package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	rt "github.com/wailsapp/wails/v2/pkg/runtime"

	"module-scanner/internal/ai"
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

	mu       sync.Mutex
	master   *masterbook.Book
	presets  *presets.Store
	aiClient *ai.Client
}

func NewApp() *App {
	store, _ := presets.Load()
	app := &App{presets: store}
	app.refreshAIClient()
	return app
}

func (a *App) refreshAIClient() {
	s := a.presets.GetSettings()
	if s.UseVLLMFallback && s.VLLMBaseURL != "" && s.VLLMModel != "" {
		a.aiClient = ai.New(s.VLLMBaseURL, s.VLLMModel)
	} else {
		a.aiClient = nil
	}
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

	var (
		barcodeHits []barcode.Hit
		barcodeErr  error
		ocrSerials  []string
		ocrErr      error
	)

	// Primary: barcode (checksum-validated, slow but reliable).
	bcHits, err := barcode.DecodeAll(data)
	if err != nil {
		barcodeErr = err
	} else {
		barcodeHits = bcHits
	}

	// Count module-length hits.
	moduleCount := 0
	for _, h := range barcodeHits {
		if len(h.Text) >= 18 {
			moduleCount++
		}
	}

	// Dominant length + prefix among barcode module serials — used to
	// filter OCR candidates. OCR has no checksum; we only accept OCR
	// results that match both the exact length AND the common prefix
	// of already-validated barcode serials.
	dominantLen := 0
	var barcodeSerials []string
	if moduleCount > 0 {
		lenCount := map[int]int{}
		for _, h := range barcodeHits {
			if len(h.Text) >= 18 {
				lenCount[len(h.Text)]++
				barcodeSerials = append(barcodeSerials, h.Text)
			}
		}
		topCount := 0
		for l, c := range lenCount {
			if c > topCount {
				topCount = c
				dominantLen = l
			}
		}
	}
	// Use prefix shared by ALL barcode serials so we don't over-filter
	// OCR candidates that happen to differ in a middle digit.
	dominantPrefix := longestCommonPrefix(barcodeSerials, len(barcodeSerials))

	// Gap-fill: if barcode didn't reach target, run PaddleOCR. OCR has
	// no checksum, so we sliding-window extract substrings of the exact
	// barcode-dominant length and require them to match the dominant
	// prefix before accepting.
	const target = 36
	if moduleCount < target && dominantLen > 0 {
		results, err := ocr.Recognize(data)
		if err != nil {
			ocrErr = err
			log.Printf("ocr recognize failed for %s: %v", filename, err)
		} else {
			raw := ocr.ExtractSerialCandidates(results, dominantLen, 40)
			seen := map[string]struct{}{}
			for _, s := range raw {
				if len(s) < dominantLen {
					continue
				}
				for i := 0; i+dominantLen <= len(s); i++ {
					sub := s[i : i+dominantLen]
					if dominantPrefix != "" && !strings.HasPrefix(sub, dominantPrefix) {
						continue
					}
					if _, dup := seen[sub]; dup {
						continue
					}
					seen[sub] = struct{}{}
					ocrSerials = append(ocrSerials, sub)
				}
			}
		}
	}

	sourceBy := map[string]string{}
	noBy := map[string]int{}
	correctedBy := map[string]string{} // serial → raw text if corrected
	for _, h := range barcodeHits {
		t := strings.TrimSpace(h.Text)
		if t == "" {
			continue
		}
		sourceBy[t] = "barcode"
		if h.Corrected {
			correctedBy[t] = h.RawText
		}
	}
	for _, s := range ocrSerials {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, has := sourceBy[s]; !has {
			sourceBy[s] = "ocr"
		}
	}

	if len(sourceBy) == 0 {
		msg := "no barcode detected"
		if barcodeErr != nil {
			msg = "no barcode: " + barcodeErr.Error()
		}
		if ocrErr != nil {
			msg += "; ocr: " + ocrErr.Error()
		}
		return []schema.ScanResult{{Filename: filename, Error: msg}}
	}

	allTexts := make([]string, 0, len(sourceBy))
	for t := range sourceBy {
		allTexts = append(allTexts, t)
	}
	modules, pallet, others := barcode.Classify(allTexts)

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
	notesFromOthers := ""
	if len(others) > 0 {
		notesFromOthers = fmt.Sprintf("기타 %d건 검출(무시): %s", len(others), others[0])
	}
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
		notes := notesFromOthers
		if corrected {
			note := fmt.Sprintf("수정됨 (원본: %s)", rawText)
			if notes != "" {
				notes = note + "; " + notes
			} else {
				notes = note
			}
		}
		out = append(out, schema.ScanResult{
			Filename:  filename,
			PhotoNo:   photoNo,
			Serial:    serial,
			Suffix:    suffix,
			Source:    src,
			PalletSN:  pallet,
			Corrected: corrected,
			RawText:   rawText,
			Notes:     notes,
		})
	}
	return out
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

func (a *App) GetSettings() presets.Settings {
	return a.presets.GetSettings()
}

type VLLMStatus struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url"`
	Model     string `json:"model"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

func (a *App) PingVLLM() VLLMStatus {
	s := a.presets.GetSettings()
	status := VLLMStatus{
		Enabled: s.UseVLLMFallback,
		URL:     s.VLLMBaseURL,
		Model:   s.VLLMModel,
	}
	if s.VLLMBaseURL == "" || s.VLLMModel == "" {
		status.Error = "URL/모델 미설정"
		return status
	}
	client := ai.New(s.VLLMBaseURL, s.VLLMModel)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	err := client.Ping(ctx)
	status.LatencyMs = time.Since(start).Milliseconds()
	if err != nil {
		status.Error = err.Error()
		return status
	}
	status.OK = true
	return status
}

func (a *App) SaveSettings(s presets.Settings) error {
	a.presets.PutSettings(s)
	err := a.presets.Save()
	a.mu.Lock()
	a.refreshAIClient()
	a.mu.Unlock()
	return err
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
