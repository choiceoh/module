package barcode

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/multi"
	"github.com/makiuchi-d/gozxing/oned"
)

type Hit struct {
	Text   string
	Format string
}

func DecodeAll(data []byte) ([]Hit, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	if hits, err := scanAtRotations(img); err == nil && len(hits) > 0 {
		return hits, nil
	}

	enhanced := imaging.AdjustContrast(imaging.Grayscale(img), 20)
	if hits, err := scanAtRotations(enhanced); err == nil && len(hits) > 0 {
		return hits, nil
	}

	return nil, fmt.Errorf("no barcode decoded")
}

func scanAtRotations(img image.Image) ([]Hit, error) {
	for _, deg := range []float64{0, 90, 180, 270} {
		rotated := img
		if deg != 0 {
			rotated = imaging.Rotate(img, deg, image.Transparent)
		}
		if hits, err := scanOne(rotated); err == nil && len(hits) > 0 {
			return hits, nil
		}
	}
	return nil, fmt.Errorf("no barcode at any rotation")
}

func scanOne(img image.Image) ([]Hit, error) {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, err
	}

	reader := multi.NewGenericMultipleBarcodeReader(oned.NewCode128Reader())
	results, err := reader.DecodeMultiple(bmp, nil)
	if err != nil {
		single, singleErr := oned.NewCode128Reader().Decode(bmp, nil)
		if singleErr != nil {
			return nil, err
		}
		return []Hit{{Text: single.GetText(), Format: single.GetBarcodeFormat().String()}}, nil
	}

	seen := map[string]bool{}
	out := make([]Hit, 0, len(results))
	for _, r := range results {
		t := r.GetText()
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, Hit{Text: t, Format: r.GetBarcodeFormat().String()})
	}
	return out, nil
}
