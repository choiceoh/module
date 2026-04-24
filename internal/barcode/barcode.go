package barcode

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

type Hit struct {
	Text   string
	Format string
}

// DecodeAll extracts Code 128 barcodes from the image.
// Tries multiple strategies to cover both stacked labels (a few barcodes
// in horizontal bands) and dense grids (packing list sheets with 30+ barcodes
// in a 2D grid).
func DecodeAll(data []byte) ([]Hit, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	hits := runStrategies(img)
	if len(hits) == 0 {
		enhanced := imaging.AdjustContrast(imaging.Grayscale(img), 20)
		hits = runStrategies(enhanced)
	}

	if len(hits) == 0 {
		return nil, fmt.Errorf("no barcode decoded")
	}
	return hits, nil
}

func runStrategies(img image.Image) []Hit {
	seen := map[string]struct{}{}
	var hits []Hit

	add := func(newHits []Hit) {
		for _, h := range newHits {
			if _, ok := seen[h.Text]; ok {
				continue
			}
			seen[h.Text] = struct{}{}
			hits = append(hits, h)
		}
	}

	strategies := [][2]int{
		{1, 1},
		{1, 4}, {1, 8}, {1, 6}, {1, 2},
		{3, 13}, {3, 12}, {3, 14}, {4, 10}, {3, 10},
		{2, 8}, {4, 13},
	}
	for _, s := range strategies {
		add(decodeGrid(img, s[0], s[1]))
	}
	return hits
}

func decodeGrid(img image.Image, cols, rows int) []Hit {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	tileW := w / cols
	tileH := h / rows
	if tileW < 60 || tileH < 20 {
		return nil
	}
	overlapX := tileW / 6
	overlapY := tileH / 6

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
			for _, deg := range []float64{0, 180} {
				target := crop
				if deg != 0 {
					target = imaging.Rotate(crop, deg, image.Transparent)
				}
				text, ok := decodeCode128(target)
				if ok {
					if _, dup := seen[text]; !dup {
						seen[text] = struct{}{}
						out = append(out, Hit{Text: text, Format: "CODE_128"})
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

// Classify groups decoded texts into module serials and a pallet serial.
// Rule: the dominant (most common) text length is assumed to be module
// serials — since a packing list has many modules but typically only one
// pallet number. Any outlier with a different length is treated as the
// pallet number. Works independent of manufacturer-specific prefixes.
//
// When only a single barcode is detected, a length threshold (>=18) is
// used to classify between a single-module photo and a pallet-only photo.
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
