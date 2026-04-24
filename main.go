package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logFile, logPath := openLogFile()
	if logFile != nil {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}
	log.Printf("=== module-scanner v%s starting (pid=%d) ===", appVersion(), os.Getpid())
	log.Printf("log file: %s", logPath)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC: %v", r)
			os.Exit(2)
		}
	}()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "모듈 시리얼 스캐너 v" + appVersion(),
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			log.Printf("OnStartup called")
			app.Startup(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			log.Printf("OnShutdown called")
		},
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Printf("wails.Run returned error: %v", err)
		log.Fatal(err)
	}
}

func openLogFile() (*os.File, string) {
	base := os.TempDir()
	if home, err := os.UserHomeDir(); err == nil {
		candidate := filepath.Join(home, ".module-scanner")
		if err := os.MkdirAll(candidate, 0o755); err == nil {
			base = candidate
		}
	}
	name := fmt.Sprintf("module-scanner-%s.log", time.Now().Format("20060102"))
	path := filepath.Join(base, name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, ""
	}
	return f, path
}
