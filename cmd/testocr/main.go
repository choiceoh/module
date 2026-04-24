package main

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"module-scanner/internal/ocr"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testocr <image>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	if err := ocr.Init(); err != nil {
		fmt.Println("init error:", err)
		os.Exit(1)
	}
	defer ocr.Cleanup()

	start := time.Now()
	results, err := ocr.Recognize(data)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Println("ocr error:", err)
		os.Exit(1)
	}

	fmt.Printf("OCR returned %d results in %s\n", len(results), elapsed)

	serialRe := regexp.MustCompile(`[A-Z0-9]{10,30}`)
	seen := map[string]struct{}{}
	for _, r := range results {
		matches := serialRe.FindAllString(r.Text, -1)
		for _, m := range matches {
			if len(m) >= 18 {
				if _, ok := seen[m]; ok {
					continue
				}
				seen[m] = struct{}{}
				fmt.Printf("  %s  (score=%.2f, box=%dx%d)\n", m, r.Score, r.Box.X1-r.Box.X0, r.Box.Y1-r.Box.Y0)
			}
		}
	}
}
