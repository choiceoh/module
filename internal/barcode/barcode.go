package barcode

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

type Hit struct {
	Text   string
	Format string
}

// DecodeAll extracts Code 128 barcodes from the image.
// Tries the full image first; if nothing found, slices the image into
// horizontal bands (1, 2, 4, 8, 6) and decodes each band separately —
// this covers cases where multiple barcodes are stacked vertically
// (e.g. multiple product labels in one photo).
func DecodeAll(data []byte) ([]Hit, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	if hits := decodeBands(img, []int{1, 4, 2, 8, 6}); len(hits) > 0 {
		return hits, nil
	}

	// Fallback: grayscale + contrast boost, then retry
	enhanced := imaging.AdjustContrast(imaging.Grayscale(img), 20)
	if hits := decodeBands(enhanced, []int{1, 4, 2, 8, 6}); len(hits) > 0 {
		return hits, nil
	}

	return nil, fmt.Errorf("no barcode decoded")
}

func decodeBands(img image.Image, bandCounts []int) []Hit {
	seen := map[string]bool{}
	var hits []Hit
	bounds := img.Bounds()
	for _, n := range bandCounts {
		bandH := bounds.Dy() / n
		if bandH < 20 {
			continue
		}
		for i := 0; i < n; i++ {
			y0 := bounds.Min.Y + i*bandH
			y1 := y0 + bandH
			if i == n-1 {
				y1 = bounds.Max.Y
			}
			crop := imaging.Crop(img, image.Rect(bounds.Min.X, y0, bounds.Max.X, y1))
			for _, rotated := range rotations(crop) {
				if text, ok := decodeCode128(rotated); ok {
					if !seen[text] {
						seen[text] = true
						hits = append(hits, Hit{Text: text, Format: "CODE_128"})
					}
					break
				}
			}
		}
	}
	return hits
}

func rotations(img image.Image) []image.Image {
	return []image.Image{
		img,
		imaging.Rotate(img, 180, image.Transparent),
	}
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
	return result.GetText(), true
}
