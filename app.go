package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	rt "github.com/wailsapp/wails/v2/pkg/runtime"

	"module-scanner/internal/barcode"
	"module-scanner/internal/excel"
	"module-scanner/internal/masterbook"
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

	hits, err := barcode.DecodeAll(data)
	if err != nil {
		return []schema.ScanResult{{Filename: filename, Error: "no barcode: " + err.Error()}}
	}
	if len(hits) == 0 {
		return []schema.ScanResult{{Filename: filename, Error: "no barcode detected"}}
	}

	texts := make([]string, 0, len(hits))
	for _, h := range hits {
		texts = append(texts, h.Text)
	}
	modules, pallet, others := barcode.Classify(texts)

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

	out := make([]schema.ScanResult, 0, len(modules))
	notesFromOthers := ""
	if len(others) > 0 {
		notesFromOthers = fmt.Sprintf("기타 %d건 검출(무시): %s", len(others), others[0])
	}
	for _, s := range modules {
		serial, suffix := splitSuffix(s)
		out = append(out, schema.ScanResult{
			Filename: filename,
			Serial:   serial,
			Suffix:   suffix,
			Source:   "barcode",
			PalletSN: pallet,
			Notes:    notesFromOthers,
		})
	}
	return out
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

func (a *App) SaveSettings(s presets.Settings) error {
	a.presets.PutSettings(s)
	return a.presets.Save()
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
