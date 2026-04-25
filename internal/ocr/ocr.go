// Package ocr runs a persistent PaddleOCR sidecar process. Models load
// once; subsequent requests only pay inference cost.
//
// Protocol (line-delimited JSON):
//   stdin:  <absolute image/pdf path>\n
//   stdout: {"raw":[...]}\n  or  {"error":"..."}\n  or  {"ready":true}\n
package ocr

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

//go:embed assets/sidecar.exe
var sidecarBlob embed.FS

var (
	initOnce    sync.Once
	initErr     error
	sidecarPath string

	sidecarMu sync.Mutex
	sidecar   *sidecarProc
)

type sidecarProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr bytes.Buffer
}

type Result struct {
	Text           string
	Score          float32
	X0, Y0, X1, Y1 int
}

type rawOutput struct {
	Raw []struct {
		Text  string  `json:"text"`
		Score float64 `json:"score"`
		X0    int     `json:"x0"`
		Y0    int     `json:"y0"`
		X1    int     `json:"x1"`
		Y1    int     `json:"y1"`
	} `json:"raw"`
	Ready bool   `json:"ready,omitempty"`
	Error string `json:"error,omitempty"`
}

// Init extracts the embedded sidecar, spawns it, and waits for the
// "ready" signal (model load complete).
func Init() error {
	initOnce.Do(func() { initErr = doInit() })
	return initErr
}

func doInit() error {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	dst := filepath.Join(dir, "module-scanner", "ocr-sidecar-v2")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("mkdir sidecar: %w", err)
	}
	sidecarPath = filepath.Join(dst, "sidecar.exe")
	if st, err := os.Stat(sidecarPath); err != nil || st.Size() == 0 {
		data, err := sidecarBlob.ReadFile("assets/sidecar.exe")
		if err != nil {
			return fmt.Errorf("read embedded sidecar: %w", err)
		}
		if err := os.WriteFile(sidecarPath, data, 0o755); err != nil {
			return fmt.Errorf("write sidecar: %w", err)
		}
	}
	return spawn()
}

func spawn() error {
	sidecarMu.Lock()
	defer sidecarMu.Unlock()
	if sidecar != nil && sidecar.cmd != nil && sidecar.cmd.ProcessState == nil && sidecar.cmd.Process != nil {
		// Already running.
		return nil
	}
	cmd := exec.Command(sidecarPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	proc := &sidecarProc{cmd: cmd, stdin: stdin}
	cmd.Stderr = &proc.stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start sidecar: %w", err)
	}
	proc.stdout = bufio.NewReaderSize(stdout, 1<<20)
	sidecar = proc

	// Wait for {"ready":true} with timeout.
	ready := make(chan error, 1)
	go func() {
		line, err := proc.stdout.ReadBytes('\n')
		if err != nil {
			ready <- err
			return
		}
		var msg rawOutput
		if err := json.Unmarshal(line, &msg); err != nil {
			ready <- err
			return
		}
		if !msg.Ready {
			ready <- fmt.Errorf("sidecar first msg not ready: %s", line)
			return
		}
		ready <- nil
	}()
	select {
	case err := <-ready:
		return err
	case <-time.After(60 * time.Second):
		return fmt.Errorf("sidecar init timeout (60s); stderr=%s", proc.stderr.String())
	}
}

// Cleanup closes stdin so the sidecar exits, then waits briefly.
func Cleanup() {
	sidecarMu.Lock()
	defer sidecarMu.Unlock()
	if sidecar == nil || sidecar.stdin == nil {
		return
	}
	_ = sidecar.stdin.Close()
	done := make(chan struct{})
	go func() { _ = sidecar.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = sidecar.cmd.Process.Kill()
	}
	sidecar = nil
}

// Recognize sends a path to the persistent sidecar and reads one JSON
// response. Serialized by mutex so concurrent callers queue up.
func Recognize(imagePath string) ([]Result, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	sidecarMu.Lock()
	defer sidecarMu.Unlock()

	if sidecar == nil {
		if err := spawn(); err != nil {
			return nil, err
		}
	}

	// Write path
	if _, err := sidecar.stdin.Write([]byte(imagePath + "\n")); err != nil {
		// Sidecar might have died; respawn once and retry.
		sidecar = nil
		if err := spawn(); err != nil {
			return nil, fmt.Errorf("respawn sidecar: %w", err)
		}
		if _, err := sidecar.stdin.Write([]byte(imagePath + "\n")); err != nil {
			return nil, fmt.Errorf("write path after respawn: %w", err)
		}
	}

	line, err := sidecar.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("sidecar read: %w (stderr=%s)", err, sidecar.stderr.String())
	}
	var parsed rawOutput
	if err := json.Unmarshal(line, &parsed); err != nil {
		return nil, fmt.Errorf("parse: %w (raw=%s)", err, line)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("sidecar: %s", parsed.Error)
	}

	results := make([]Result, 0, len(parsed.Raw))
	for _, r := range parsed.Raw {
		results = append(results, Result{
			Text:  r.Text,
			Score: float32(r.Score),
			X0:    r.X0, Y0: r.Y0, X1: r.X1, Y1: r.Y1,
		})
	}
	return results, nil
}

// RecognizeBytes writes image bytes to a temp file and runs Recognize.
// For PDFs the caller should use Recognize directly since extension matters.
func RecognizeBytes(data []byte) ([]Result, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	// Detect PDF by magic bytes to use proper extension.
	ext := ".png"
	if len(data) >= 4 && string(data[:4]) == "%PDF" {
		ext = ".pdf"
	} else if len(data) >= 3 && string(data[:3]) == "\xff\xd8\xff" {
		ext = ".jpg"
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	tmp, err := os.CreateTemp(dir, "ocr-input-*"+ext)
	if err != nil {
		return nil, err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return nil, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	return Recognize(tmp.Name())
}
