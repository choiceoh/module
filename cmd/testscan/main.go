package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"module-scanner/internal/ai"
	"module-scanner/internal/barcode"
)

func main() {
	var (
		vllmURL   = flag.String("vllm-url", "", "vLLM base URL (e.g. https://host/v1). Empty = barcode only.")
		vllmModel = flag.String("vllm-model", "gemma4", "vLLM model name")
		verbose   = flag.Bool("v", false, "print all raw hits (not just classified modules)")
	)
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "usage: testscan [-vllm-url URL -vllm-model NAME -v] <image>...")
		os.Exit(2)
	}

	var aiClient *ai.Client
	if *vllmURL != "" {
		aiClient = ai.New(*vllmURL, *vllmModel)
	}

	for _, path := range paths {
		testOne(path, aiClient, *verbose)
		fmt.Println()
	}
}

func testOne(path string, aiClient *ai.Client, verbose bool) {
	fmt.Printf("=== %s ===\n", path)
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("  read error: %v\n", err)
		return
	}
	fmt.Printf("  size: %d bytes\n", len(data))

	// Sequential: barcode first; only call VLM with "find missing" prompt
	// if barcode is below target. This skips VLM entirely when barcode
	// already got everything.
	start := time.Now()
	bcHits, bcErr := barcode.DecodeAll(data)
	bcTime := time.Since(start)

	var (
		vllmEntries []ai.ModuleEntry
		vllmPallet  string
		vllmErr     error
		vllmTime    time.Duration
	)

	// Count only module-length hits (pallet serials are much shorter).
	moduleCount := 0
	for _, h := range bcHits {
		if len(h.Text) >= 18 {
			moduleCount++
		}
	}
	const target = 36 // expected modules per pallet; tunable
	if aiClient != nil && moduleCount < target {
		known := make([]string, 0, len(bcHits))
		for _, h := range bcHits {
			known = append(known, h.Text)
		}
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		ext, err := aiClient.ExtractMissingSerials(ctx, data, "image/png", known)
		cancel()
		vllmTime = time.Since(start)
		if err != nil {
			vllmErr = err
		} else {
			vllmEntries = ext.Modules
			vllmPallet = ext.PalletSN
		}
	}

	fmt.Printf("  barcode: %d raw hits in %s (err=%v)\n", len(bcHits), bcTime.Round(time.Millisecond), bcErr)
	if aiClient != nil {
		fmt.Printf("  vllm:    %d entries, pallet=%q, in %s (err=%v)\n", len(vllmEntries), vllmPallet, vllmTime.Round(time.Millisecond), vllmErr)
	}

	sourceBy := map[string]string{}
	noBy := map[string]int{}
	for _, h := range bcHits {
		sourceBy[h.Text] = "barcode"
	}
	for _, e := range vllmEntries {
		s := strings.TrimSpace(e.Serial)
		if s == "" {
			continue
		}
		if _, has := sourceBy[s]; !has {
			sourceBy[s] = "vllm"
		}
		if _, has := noBy[s]; !has && e.No > 0 {
			noBy[s] = e.No
		}
	}

	allTexts := make([]string, 0, len(sourceBy))
	for t := range sourceBy {
		allTexts = append(allTexts, t)
	}
	modules, pallet, others := barcode.Classify(allTexts)
	if pallet == "" && vllmPallet != "" {
		pallet = strings.TrimSpace(vllmPallet)
	}

	sort.SliceStable(modules, func(i, j int) bool {
		ni, oi := noBy[modules[i]]
		nj, oj := noBy[modules[j]]
		switch {
		case oi && oj:
			return ni < nj
		case oi:
			return true
		case oj:
			return false
		default:
			return modules[i] < modules[j]
		}
	})

	fmt.Printf("  classified: pallet=%q modules=%d others=%d\n", pallet, len(modules), len(others))

	sourceCounts := map[string]int{}
	for _, m := range modules {
		sourceCounts[sourceBy[m]]++
	}
	fmt.Printf("  by source: %v\n", sourceCounts)

	for i, m := range modules {
		tag := sourceBy[m]
		no := noBy[m]
		noStr := ""
		if no > 0 {
			noStr = fmt.Sprintf(" no=%d", no)
		}
		fmt.Printf("    %2d  %s  [%s]%s\n", i+1, m, tag, noStr)
	}

	if verbose {
		fmt.Println("  raw barcode hits:")
		for _, h := range bcHits {
			note := ""
			if h.Corrected {
				note = fmt.Sprintf("  (corrected from %q)", h.RawText)
			}
			fmt.Printf("    %s%s\n", h.Text, note)
		}
	}
}

var _ = http.StatusOK
