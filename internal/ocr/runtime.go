// Package ocr is a pure-Go port of the PaddleOCR detection + recognition
// pipeline (PP-OCRv4 mobile models), running via onnxruntime.
//
// It provides a single public function, Recognize, that takes an image
// and returns all recognized text lines with their bounding boxes.
// Used as a gap-fill for the barcode decoder: when a Code 128 reader
// can't decode a barcode due to blur or angle, the human-readable
// serial text printed beneath it is almost always legible by OCR.
package ocr

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

//go:embed assets/onnxruntime.dll assets/onnxruntime_providers_shared.dll assets/ch_PP-OCRv4_det_infer.onnx assets/ch_PP-OCRv4_rec_infer.onnx assets/dict.txt
var assets embed.FS

var (
	initOnce    sync.Once
	initErr     error
	assetDir    string
	detPath     string
	recPath     string
	dictChars   []string // includes leading blank at index 0 for CTC
)

// Init extracts embedded ONNX assets to a temp dir and initializes the
// ONNX runtime. Safe to call multiple times; the first caller does the
// work and subsequent callers return the cached result.
func Init() error {
	initOnce.Do(func() {
		initErr = doInit()
	})
	return initErr
}

func doInit() error {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	assetDir = filepath.Join(dir, "module-scanner", "ocr-assets-v1")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return fmt.Errorf("mkdir assets: %w", err)
	}

	names := []string{
		"onnxruntime.dll",
		"onnxruntime_providers_shared.dll",
		"ch_PP-OCRv4_det_infer.onnx",
		"ch_PP-OCRv4_rec_infer.onnx",
		"dict.txt",
	}
	for _, n := range names {
		dst := filepath.Join(assetDir, n)
		if st, err := os.Stat(dst); err == nil && st.Size() > 0 {
			continue // already extracted
		}
		data, err := assets.ReadFile("assets/" + n)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", n, err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}

	detPath = filepath.Join(assetDir, "ch_PP-OCRv4_det_infer.onnx")
	recPath = filepath.Join(assetDir, "ch_PP-OCRv4_rec_infer.onnx")

	dictData, err := assets.ReadFile("assets/dict.txt")
	if err != nil {
		return fmt.Errorf("read dict: %w", err)
	}
	// PPOCR CTC convention: index 0 = blank, indices 1..N = dict chars,
	// index N+1 = end-of-sequence blank. We store with leading blank
	// at index 0 so that output indices map directly.
	dictChars = []string{""}
	line := make([]byte, 0, 8)
	for _, b := range dictData {
		if b == '\n' {
			dictChars = append(dictChars, string(line))
			line = line[:0]
			continue
		}
		if b == '\r' {
			continue
		}
		line = append(line, b)
	}
	if len(line) > 0 {
		dictChars = append(dictChars, string(line))
	}
	// PPOCR appends a space; total classes must equal model output.
	dictChars = append(dictChars, " ")

	ort.SetSharedLibraryPath(filepath.Join(assetDir, "onnxruntime.dll"))
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("ort init: %w", err)
	}
	return nil
}

// Cleanup releases the ONNX runtime. Call on app shutdown.
func Cleanup() {
	_ = ort.DestroyEnvironment()
}
