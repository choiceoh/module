// Package barcode decodes Code 128 serial numbers from photos of pallet
// packing-list sheets (Jinko, Trina, etc.) as well as simpler single- or
// few-barcode labels.
//
// Pipeline (for each input image):
//
//  1. Decode & grayscale.
//  2. Row detection — projection-based search for horizontal strips with
//     many black/white transitions (characteristic of 1D barcodes).
//  3. Column split — each detected row is split into N equal tiles.
//  4. Per-cell decode — top-55% crop (barcode only, skipping the printed
//     text), upscale 2x, Sauvola adaptive threshold. Multi-try: TRY_HARDER
//     and PURE_BARCODE hints, small rotations (-6, -3, 0, 3, 6 deg).
//  5. Fallback — if a cell fails, fall back to whole-image grid scans and
//     iterative mask-decode on row strips.
//  6. Post-process — drop short spurious reads, repair prefix outliers,
//     classify into pallet SN / module SNs.
//
// All per-cell work is executed on a bounded worker pool (GOMAXPROCS),
// so 3000x3000 photos with 36 cells decode in a few seconds.
package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"runtime"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

type Hit struct {
	Text      string
	Format    string
	Corrected bool   // prefix was repaired from the raw decode
	RawText   string // original decode before correction (empty if unchanged)
}

// DecodeAll is the package entry point. See file-level doc for the pipeline.
func DecodeAll(data []byte) ([]Hit, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	// Precompute the expensive preprocessing variants exactly once and
	// share crops of them across all decode goroutines. This is the
	// single biggest speed win.
	var gray, otsu, sauvola image.Image
	var preWG sync.WaitGroup
	preWG.Add(3)
	go func() { defer preWG.Done(); gray = imaging.Grayscale(img) }()
	go func() { defer preWG.Done(); otsu = otsuBinarize(img) }()
	go func() { defer preWG.Done(); sauvola = sauvolaBinarize(img, 25) }()
	preWG.Wait()

	raw := collectHits(img, gray, otsu, sauvola)
	if len(raw) == 0 {
		return nil, fmt.Errorf("no barcode decoded")
	}
	return repairOutliersByPrefix(dedupe(raw)), nil
}

// collectHits runs the cell-based pipeline on the full image plus on
// per-column slices (to handle uneven lighting where a global threshold
// misses one column), and a whole-image grid fallback for stacked-label
// photos. Everything runs in parallel with shared-memory deduplication.
func collectHits(img, gray, otsu, sauvola image.Image) []Hit {
	var seenMu sync.Mutex
	seen := map[string]struct{}{}
	var hits []Hit
	add := func(h Hit) {
		seenMu.Lock()
		defer seenMu.Unlock()
		if _, ok := seen[h.Text]; ok {
			return
		}
		seen[h.Text] = struct{}{}
		hits = append(hits, h)
	}

	var wg sync.WaitGroup

	// 1. Full-image cell scan with all three precomputed variants.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, h := range scanCells(img, otsu, sauvola) {
			add(h)
		}
	}()

	// 2. Per-column-slice cell scan — catches middle-column barcodes
	//    that global Otsu washes out due to local brightness differences.
	bounds := img.Bounds()
	w := bounds.Dx()
	colRanges := [][2]int{
		{0, w * 45 / 100},
		{w * 28 / 100, w * 75 / 100},
		{w * 55 / 100, w},
	}
	for _, cr := range colRanges {
		lo, hi := cr[0], cr[1]
		wg.Add(1)
		go func(lo, hi int) {
			defer wg.Done()
			slice := imaging.Crop(img, image.Rect(bounds.Min.X+lo, bounds.Min.Y, bounds.Min.X+hi, bounds.Max.Y))
			sliceOtsu := otsuBinarize(slice)
			for _, h := range scanCells(slice, sliceOtsu, nil) {
				add(h)
			}
		}(lo, hi)
	}

	// 3. Whole-image grid fallback (single-row label photos).
	wg.Add(1)
	go func() {
		defer wg.Done()
		contrast := imaging.AdjustContrast(gray, 40)
		for _, src := range []image.Image{img, otsu, contrast} {
			for _, g := range [][2]int{{1, 1}, {1, 4}, {1, 8}} {
				for _, h := range decodeGrid(src, g[0], g[1]) {
					add(h)
				}
			}
		}
	}()

	wg.Wait()
	return hits
}

// scanCells detects barcode row regions, splits each into 3/4/6 column
// tiles, and decodes each cell in parallel on a worker pool. Cells are
// cropped from the original image and a precomputed Sauvola variant so
// no per-cell binarization is needed.
func scanCells(img, otsu, sauvola image.Image) []Hit {
	rows := detectBarcodeRows(otsu, 3)
	if len(rows) == 0 {
		return nil
	}

	bounds := img.Bounds()
	w := bounds.Dx()

	type cell struct {
		x0, y0, x1, y1 int
	}
	var cells []cell

	colCounts := []int{3, 4, 6}
	for _, cols := range colCounts {
		tileW := w / cols
		overlapX := tileW / 3
		for _, r := range rows {
			for c := 0; c < cols; c++ {
				x0 := c*tileW - overlapX
				x1 := (c+1)*tileW + overlapX
				if x0 < 0 {
					x0 = 0
				}
				if x1 > w {
					x1 = w
				}
				cells = append(cells, cell{
					x0: bounds.Min.X + x0,
					y0: bounds.Min.Y + r[0],
					x1: bounds.Min.X + x1,
					y1: bounds.Min.Y + r[1],
				})
			}
		}
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 2 {
		workerCount = 2
	}

	cellCh := make(chan cell, len(cells))
	hitCh := make(chan Hit, len(cells))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range cellCh {
				rect := image.Rect(c.x0, c.y0, c.x1, c.y1)
				crop := imaging.Crop(img, rect)
				var sauvolaCrop image.Image
				if sauvola != nil {
					sauvolaCrop = imaging.Crop(sauvola, rect)
				}
				if text, ok := decodeCell(crop, sauvolaCrop); ok {
					hitCh <- Hit{Text: text, Format: "CODE_128"}
				}
			}
		}()
	}
	for _, c := range cells {
		cellCh <- c
	}
	close(cellCh)

	go func() {
		wg.Wait()
		close(hitCh)
	}()

	seen := map[string]struct{}{}
	var hits []Hit
	for h := range hitCh {
		if _, ok := seen[h.Text]; ok {
			continue
		}
		seen[h.Text] = struct{}{}
		hits = append(hits, h)
	}
	return hits
}

// decodeCell tries raw → top-55%-crop → Sauvola-crop variants, with 2x
// upscale and a few small rotations. All binarization is pre-computed;
// this function just crops and rotates.
func decodeCell(crop, sauvolaCrop image.Image) (string, bool) {
	b := crop.Bounds()
	if b.Dx() < 60 || b.Dy() < 20 {
		return "", false
	}

	candidates := []image.Image{crop}
	if sauvolaCrop != nil {
		candidates = append(candidates, sauvolaCrop)
	}
	if b.Dy() >= 40 {
		top := imaging.Crop(crop, image.Rect(0, 0, b.Dx(), b.Dy()*55/100))
		candidates = append(candidates, top)
		if sauvolaCrop != nil {
			sb := sauvolaCrop.Bounds()
			topS := imaging.Crop(sauvolaCrop, image.Rect(0, 0, sb.Dx(), sb.Dy()*55/100))
			candidates = append(candidates, topS)
		}
	}

	for _, cand := range candidates {
		scaled := cand
		cw := cand.Bounds().Dx()
		if cw < 1200 {
			scaled = imaging.Resize(cand, cw*2, 0, imaging.Lanczos)
		}
		for _, deg := range []float64{0, -3, 3, -6, 6} {
			attempt := scaled
			if deg != 0 {
				attempt = imaging.Rotate(scaled, deg, image.Transparent)
			}
			if text, ok := decodeCode128(attempt); ok {
				return text, true
			}
		}
	}
	return "", false
}

// decodeGrid is the fallback whole-image grid scanner used for
// single-label or stacked-label photos where cell detection doesn't apply.
func decodeGrid(img image.Image, cols, rows int) []Hit {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	tileW := w / cols
	tileH := h / rows
	if tileW < 60 || tileH < 20 {
		return nil
	}
	overlapX := tileW / 3
	overlapY := tileH / 2

	seen := map[string]struct{}{}
	var out []Hit

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0 := c*tileW - overlapX
			y0 := r*tileH - overlapY
			x1 := (c+1)*tileW + overlapX
			y1 := (r+1)*tileH + overlapY
			if x0 < bounds.Min.X {
				x0 = bounds.Min.X
			}
			if y0 < bounds.Min.Y {
				y0 = bounds.Min.Y
			}
			if x1 > bounds.Max.X {
				x1 = bounds.Max.X
			}
			if y1 > bounds.Max.Y {
				y1 = bounds.Max.Y
			}
			crop := imaging.Crop(img, image.Rect(x0, y0, x1, y1))
			if text, ok := decodeCell(crop, nil); ok {
				if _, dup := seen[text]; !dup {
					seen[text] = struct{}{}
					out = append(out, Hit{Text: text, Format: "CODE_128"})
				}
			}
		}
	}
	return out
}

// decodeCode128 calls the gozxing Code 128 reader twice — once with
// TRY_HARDER and once with PURE_BARCODE — and accepts only results that
// match the looksLikeSerial shape.
func decodeCode128(img image.Image) (string, bool) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", false
	}
	for _, hints := range []map[gozxing.DecodeHintType]interface{}{
		{gozxing.DecodeHintType_TRY_HARDER: true},
		{gozxing.DecodeHintType_PURE_BARCODE: true},
	} {
		result, err := oned.NewCode128Reader().Decode(bmp, hints)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(result.GetText())
		if looksLikeSerial(text) {
			return text, true
		}
	}
	return "", false
}

// looksLikeSerial accepts only uppercase alphanumeric strings. Code 128
// returns whatever bars happen to encode; real module/pallet serials are
// uppercase hex-ish, so anything with lowercase letters, underscores,
// asterisks, colons, slashes, etc. is a spurious decode of a region that
// happened to pass checksum by coincidence.
func looksLikeSerial(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 6 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '|' || r == ' ':
		default:
			return false
		}
	}
	return true
}

// detectBarcodeRows scans the binarized image and returns Y ranges that
// look like barcode rows — rows with many black/white transitions.
func detectBarcodeRows(bin image.Image, padRatio int) [][2]int {
	bounds := bin.Bounds()
	h := bounds.Dy()
	w := bounds.Dx()
	if h < 80 || w < 80 {
		return nil
	}

	transitions := make([]int, h)
	for y := 0; y < h; y++ {
		prev := uint8(255)
		count := 0
		for x := 0; x < w; x++ {
			r, _, _, _ := bin.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			cur := uint8(r >> 8)
			if (cur < 128) != (prev < 128) {
				count++
			}
			prev = cur
		}
		transitions[y] = count
	}

	minTransitions := w / 40
	if minTransitions < 30 {
		minTransitions = 30
	}

	var regions [][2]int
	inRow := false
	start := 0
	for y := 0; y < h; y++ {
		high := transitions[y] >= minTransitions
		if high && !inRow {
			start = y
			inRow = true
		} else if !high && inRow {
			if y-start >= 15 {
				pad := (y - start) / padRatio
				y0 := start - pad
				y1 := y + pad
				if y0 < 0 {
					y0 = 0
				}
				if y1 > h {
					y1 = h
				}
				regions = append(regions, [2]int{y0, y1})
			}
			inRow = false
		}
	}
	if inRow && h-start >= 15 {
		regions = append(regions, [2]int{start, h})
	}
	return regions
}

// Classify groups decoded texts into module serials and a pallet serial
// by the "majority length" rule: a packing list carries many modules
// (same length) and typically a single pallet code (different length).
func Classify(texts []string) (modules []string, pallet string, others []string) {
	if len(texts) == 0 {
		return
	}
	if len(texts) == 1 {
		t := texts[0]
		if len(t) >= 18 {
			return []string{t}, "", nil
		}
		return nil, t, nil
	}

	byLen := map[int]int{}
	for _, t := range texts {
		byLen[len(t)]++
	}
	var domLen, domCount int
	for l, c := range byLen {
		if c > domCount || (c == domCount && l > domLen) {
			domLen = l
			domCount = c
		}
	}

	for _, t := range texts {
		if len(t) == domLen {
			modules = append(modules, t)
			continue
		}
		if pallet == "" {
			pallet = t
		} else {
			others = append(others, t)
		}
	}
	return
}

// repairOutliersByPrefix patches hits whose prefix deviates from the
// majority. Hits that map to an existing canonical one after repair are
// dropped as duplicates. Non-dedup hits are kept with Corrected=true
// so the UI can show a "needs review" marker.
func repairOutliersByPrefix(hits []Hit) []Hit {
	if len(hits) < 4 {
		return hits
	}
	byLen := map[int][]Hit{}
	for _, h := range hits {
		byLen[len(h.Text)] = append(byLen[len(h.Text)], h)
	}

	var result []Hit
	for _, group := range byLen {
		if len(group) < 4 {
			result = append(result, group...)
			continue
		}
		prefix := dominantPrefix(group)
		if prefix == "" {
			result = append(result, group...)
			continue
		}
		canonical := map[string]bool{}
		for _, h := range group {
			if strings.HasPrefix(h.Text, prefix) {
				canonical[h.Text] = true
			}
		}
		for _, h := range group {
			if strings.HasPrefix(h.Text, prefix) {
				result = append(result, h)
				continue
			}
			repaired := prefix + h.Text[len(prefix):]
			if len(repaired) != len(h.Text) {
				result = append(result, Hit{Text: h.Text, Format: h.Format, Corrected: true, RawText: h.Text})
				continue
			}
			if canonical[repaired] {
				continue
			}
			result = append(result, Hit{
				Text:      repaired,
				Format:    h.Format,
				Corrected: true,
				RawText:   h.Text,
			})
			canonical[repaired] = true
		}
	}
	return result
}

func dominantPrefix(group []Hit) string {
	minMajority := (len(group) + 1) / 2
	minLen := len(group[0].Text)
	for _, h := range group {
		if len(h.Text) < minLen {
			minLen = len(h.Text)
		}
	}
	var best string
	for l := 4; l <= minLen-4; l++ {
		counts := map[string]int{}
		for _, h := range group {
			counts[h.Text[:l]]++
		}
		top := 0
		topPrefix := ""
		for p, c := range counts {
			if c > top {
				top = c
				topPrefix = p
			}
		}
		if top >= minMajority {
			best = topPrefix
		} else {
			break
		}
	}
	return best
}

func dedupe(hits []Hit) []Hit {
	seen := map[string]struct{}{}
	var out []Hit
	for _, h := range hits {
		if _, ok := seen[h.Text]; ok {
			continue
		}
		seen[h.Text] = struct{}{}
		out = append(out, h)
	}
	return out
}

// otsuBinarize applies grayscale + global Otsu threshold.
func otsuBinarize(src image.Image) image.Image {
	gray := imaging.Grayscale(src)
	bounds := gray.Bounds()

	var hist [256]int
	total := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := gray.At(x, y).RGBA()
			hist[r>>8]++
			total++
		}
	}
	if total == 0 {
		return gray
	}

	var sum float64
	for i, c := range hist {
		sum += float64(i) * float64(c)
	}

	var (
		sumB      float64
		wB        int
		maxVar    float64
		threshold uint8 = 128
	)
	for i := 0; i < 256; i++ {
		wB += hist[i]
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(i) * float64(hist[i])
		mB := sumB / float64(wB)
		mF := (sum - sumB) / float64(wF)
		variance := float64(wB) * float64(wF) * (mB - mF) * (mB - mF)
		if variance > maxVar {
			maxVar = variance
			threshold = uint8(i)
		}
	}

	bin := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, _, _, _ := gray.At(x, y).RGBA()
			if uint8(r>>8) > threshold {
				bin.Set(x, y, color.White)
			} else {
				bin.Set(x, y, color.Black)
			}
		}
	}
	return bin
}

// sauvolaBinarize applies Sauvola local adaptive thresholding in O(w*h)
// via integral images. Chooses per-pixel thresholds based on a local
// window's mean and std deviation, so uneven lighting across a photo
// doesn't black out or wash out parts of the barcode.
func sauvolaBinarize(src image.Image, windowSize int) image.Image {
	gray := imaging.Grayscale(src)
	bounds := gray.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 10 || h < 10 {
		return gray
	}

	integral := make([]int64, (w+1)*(h+1))
	sqIntegral := make([]int64, (w+1)*(h+1))
	pixels := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, _, _, _ := gray.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			v := uint8(r >> 8)
			pixels[y*w+x] = v
			v64 := int64(v)
			i := (y+1)*(w+1) + (x + 1)
			integral[i] = v64 + integral[i-1] + integral[i-(w+1)] - integral[i-(w+1)-1]
			sqIntegral[i] = v64*v64 + sqIntegral[i-1] + sqIntegral[i-(w+1)] - sqIntegral[i-(w+1)-1]
		}
	}

	half := windowSize / 2
	k := 0.2
	R := 128.0

	result := image.NewRGBA(bounds)
	for y := 0; y < h; y++ {
		y0 := y - half
		if y0 < 0 {
			y0 = 0
		}
		y1 := y + half + 1
		if y1 > h {
			y1 = h
		}
		for x := 0; x < w; x++ {
			x0 := x - half
			if x0 < 0 {
				x0 = 0
			}
			x1 := x + half + 1
			if x1 > w {
				x1 = w
			}
			area := int64((y1 - y0) * (x1 - x0))
			sum := integral[y1*(w+1)+x1] - integral[y0*(w+1)+x1] - integral[y1*(w+1)+x0] + integral[y0*(w+1)+x0]
			sqSum := sqIntegral[y1*(w+1)+x1] - sqIntegral[y0*(w+1)+x1] - sqIntegral[y1*(w+1)+x0] + sqIntegral[y0*(w+1)+x0]
			mean := float64(sum) / float64(area)
			variance := float64(sqSum)/float64(area) - mean*mean
			if variance < 0 {
				variance = 0
			}
			stddev := math.Sqrt(variance)
			threshold := mean * (1 + k*(stddev/R-1))

			pix := pixels[y*w+x]
			cx := bounds.Min.X + x
			cy := bounds.Min.Y + y
			if float64(pix) > threshold {
				result.Set(cx, cy, color.RGBA{255, 255, 255, 255})
			} else {
				result.Set(cx, cy, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return result
}

// draw is kept as an explicit import so future iterative-mask strategies
// can still use it if re-introduced.
var _ = draw.Src
