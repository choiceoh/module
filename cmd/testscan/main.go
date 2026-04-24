package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"module-scanner/internal/barcode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: testscan <image>...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		testOne(path)
		fmt.Println()
	}
}

func testOne(path string) {
	fmt.Printf("=== %s ===\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  read error: %v\n", err)
		return
	}
	fmt.Printf("  size: %d bytes\n", len(data))

	start := time.Now()
	hits, err := barcode.DecodeAll(data)
	elapsed := time.Since(start)
	fmt.Printf("  decode time: %s\n", elapsed)

	if err != nil {
		fmt.Printf("  decode error: %v\n", err)
		return
	}

	texts := make([]string, 0, len(hits))
	for _, h := range hits {
		texts = append(texts, h.Text)
	}
	modules, pallet, others := barcode.Classify(texts)

	fmt.Printf("  total decoded: %d\n", len(hits))
	fmt.Printf("  pallet: %q\n", pallet)
	fmt.Printf("  modules: %d\n", len(modules))
	fmt.Printf("  others: %d %v\n", len(others), others)

	sort.Strings(modules)
	// Index hits by text to report Corrected/RawText info.
	hitByText := map[string]barcode.Hit{}
	for _, h := range hits {
		hitByText[h.Text] = h
	}
	for i, m := range modules {
		suffix := ""
		if h, ok := hitByText[m]; ok && h.Corrected {
			suffix = fmt.Sprintf("  [corrected from %q]", h.RawText)
		}
		if strings.Contains(m, "|") || strings.Contains(m, " ") {
			suffix += " (has separator)"
		}
		fmt.Printf("    %2d  %s%s\n", i+1, m, suffix)
	}
}
