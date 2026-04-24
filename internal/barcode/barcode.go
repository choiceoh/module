package barcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

type Hit struct {
	Text   string
	Format string
}

// DecodeAll extracts Code 128 barcodes from the image.
//   - Preprocessing: grayscale + Otsu binarization (document-scanner style)
//   - Strategies: whole-image + multiple grid sizes (handles single labels
//     and dense packing lists)
//   - Parallel execution across strategies (one goroutine per strategy)
//   - Per-tile rotation probes (-5, 0, 5 deg) to handle slight tilt
func DecodeAll(data []byte) ([]Hit, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	variants := []image.Image{
		img,
		otsuBinarize(img),
		imaging.AdjustContrast(imaging.Grayscale(img), 40),
	}

	strategies := [][2]int{
		{1, 1},
		{3, 13}, {3, 14}, {3, 15},
		{4, 13}, {4, 14},
		{2, 12},
	}

	type hit struct{ text string }
	out := make(chan hit, 4096)

	var wg sync.WaitGroup
	for _, v := range variants {
		for _, s := range strategies {
			wg.Add(1)
			go func(src image.Image, cols, rows int) {
				defer wg.Done()
				for _, t := range decodeGrid(src, cols, rows) {
					out <- hit{t}
				}
			}(v, s[0], s[1])
		}
	}
	go func() { wg.Wait(); close(out) }()

	seen := map[string]struct{}{}
	var hits []Hit
	for h := range out {
		if _, ok := seen[h.text]; ok {
			continue
		}
		seen[h.text] = struct{}{}
		hits = append(hits, Hit{Text: h.text, Format: "CODE_128"})
	}

	if len(hits) == 0 {
		return nil, fmt.Errorf("no barcode decoded")
	}
	return hits, nil
}

func decodeGrid(img image.Image, cols, rows int) []string {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	tileW := w / cols
	tileH := h / rows
	if tileW < 60 || tileH < 20 {
		return nil
	}
	overlapX := tileW / 4
	overlapY := tileH / 3

	seen := map[string]struct{}{}
	var out []string

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

			target := crop
			cw := crop.Bounds().Dx()
			if cw > 0 && cw < 500 {
				target = imaging.Resize(crop, cw*2, 0, imaging.Lanczos)
			}

			for _, deg := range []float64{0, -5, 5} {
				attempt := target
				if deg != 0 {
					attempt = imaging.Rotate(target, deg, image.Transparent)
				}
				if text, ok := decodeCode128(attempt); ok {
					if _, dup := seen[text]; !dup {
						seen[text] = struct{}{}
						out = append(out, text)
					}
					break
				}
			}
		}
	}
	return out
}

func decodeCode128(img image.Image) (string, bool) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", false
	}
	hints := map[gozxing.DecodeHintType]interface{}{
		gozxing.DecodeHintType_TRY_HARDER: true,
	}
	result, err := oned.NewCode128Reader().Decode(bmp, hints)
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(result.GetText()), true
}

// otsuBinarize applies grayscale + global Otsu threshold.
// Produces a pure black/white image that helps Code 128 decode on noisy
// photos (shadows, paper texture, mild blur).
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

// Classify groups decoded texts into module serials and a pallet serial
// by the "majority length" rule: a packing list carries many modules
// (same length) and typically a single pallet code (different length).
//
// Works across vendors (Jinko E2FXJ… 24-char, Trina R0126… 15-char, etc.)
// without hardcoded prefixes.
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
