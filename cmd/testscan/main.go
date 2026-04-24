package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"module-scanner/internal/barcode"
	"module-scanner/internal/ocr"
)

func main() {
	var (
		useOCR  = flag.Bool("ocr", true, "Run PaddleOCR gap-fill when barcode is below target")
		verbose = flag.Bool("v", false, "print all raw hits")
	)
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "usage: testscan [-ocr] [-v] <image>...")
		os.Exit(2)
	}

	if *useOCR {
		if err := ocr.Init(); err != nil {
			fmt.Fprintln(os.Stderr, "ocr init:", err)
			os.Exit(1)
		}
		defer ocr.Cleanup()
	}

	for _, p := range paths {
		testOne(p, *useOCR, *verbose)
		fmt.Println()
	}
}

func testOne(path string, useOCR, verbose bool) {
	fmt.Printf("=== %s ===\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  read error: %v\n", err)
		return
	}
	fmt.Printf("  size: %d bytes\n", len(data))

	start := time.Now()
	bcHits, bcErr := barcode.DecodeAll(data)
	bcTime := time.Since(start)

	moduleLenCount := map[int]int{}
	for _, h := range bcHits {
		if len(h.Text) >= 18 {
			moduleLenCount[len(h.Text)]++
		}
	}
	dominantLen, topCount := 0, 0
	for l, c := range moduleLenCount {
		if c > topCount {
			topCount = c
			dominantLen = l
		}
	}

	var (
		ocrSerials []string
		ocrTime    time.Duration
		ocrErr     error
	)
	// Compute dominant prefix from barcode-verified serials
	var bcSerials []string
	for _, h := range bcHits {
		if len(h.Text) >= 18 {
			bcSerials = append(bcSerials, h.Text)
		}
	}
	prefix := longestCommonPrefix(bcSerials, len(bcSerials))

	const target = 36
	if useOCR && topCount < target && dominantLen > 0 {
		t := time.Now()
		results, err := ocr.Recognize(data)
		ocrTime = time.Since(t)
		if err != nil {
			ocrErr = err
		} else {
			raw := ocr.ExtractSerialCandidates(results, dominantLen, 40)
			if verbose {
				fmt.Printf("  ocr raw candidates: %d, prefix=%q\n", len(raw), prefix)
			}
			seen := map[string]struct{}{}
			for _, s := range raw {
				for i := 0; i+dominantLen <= len(s); i++ {
					sub := s[i : i+dominantLen]
					if prefix != "" && !strings.HasPrefix(sub, prefix) {
						continue
					}
					if _, dup := seen[sub]; dup {
						continue
					}
					seen[sub] = struct{}{}
					ocrSerials = append(ocrSerials, sub)
					if verbose && (strings.Contains(sub, "552") || strings.Contains(sub, "171")) {
						fmt.Printf("    accepted %q from raw %q\n", sub, s)
					}
				}
			}
		}
	}

	fmt.Printf("  barcode: %d raw hits in %s (err=%v)\n", len(bcHits), bcTime.Round(time.Millisecond), bcErr)
	fmt.Printf("  dominant-len: %d (%d serials)\n", dominantLen, topCount)
	if useOCR {
		fmt.Printf("  ocr:     %d length-matched serials in %s (err=%v)\n", len(ocrSerials), ocrTime.Round(time.Millisecond), ocrErr)
	}

	sourceBy := map[string]string{}
	for _, h := range bcHits {
		sourceBy[h.Text] = "barcode"
	}
	for _, s := range ocrSerials {
		if _, has := sourceBy[s]; !has {
			sourceBy[s] = "ocr"
		}
	}

	allTexts := make([]string, 0, len(sourceBy))
	for t := range sourceBy {
		allTexts = append(allTexts, t)
	}
	modules, pallet, others := barcode.Classify(allTexts)

	sort.Strings(modules)

	fmt.Printf("  classified: pallet=%q modules=%d others=%d\n", pallet, len(modules), len(others))
	sc := map[string]int{}
	for _, m := range modules {
		sc[sourceBy[m]]++
	}
	fmt.Printf("  by source: %v\n", sc)

	for i, m := range modules {
		fmt.Printf("    %2d  %s  [%s]\n", i+1, m, sourceBy[m])
	}

	if verbose {
		fmt.Println("  raw barcode:", len(bcHits))
		for _, h := range bcHits {
			note := ""
			if h.Corrected {
				note = fmt.Sprintf(" (corrected from %q)", h.RawText)
			}
			fmt.Printf("    %s%s\n", h.Text, note)
		}
		fmt.Println("  raw ocr length-filtered:", len(ocrSerials))
		for _, s := range ocrSerials {
			fmt.Printf("    %s\n", s)
		}
	}
	_ = strings.TrimSpace
}

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
