package ocr

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/disintegration/imaging"
)

// Result is one recognized text line with its bounding box in the
// original image's coordinate space.
type Result struct {
	Text  string
	Score float32
	Box   Box
}

// Recognize runs the full PP-OCRv4 pipeline on the given image bytes
// and returns all recognized text lines.
//
// Call Init() once before the first use.
func Recognize(data []byte) ([]Result, error) {
	if err := Init(); err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	boxes, err := detectTextBoxes(img)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}

	results := make([]Result, 0, len(boxes))
	for _, b := range boxes {
		crop := imaging.Crop(img, image.Rect(b.X0, b.Y0, b.X1, b.Y1))
		text, score, err := recognizeText(crop)
		if err != nil {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		results = append(results, Result{Text: text, Score: score, Box: b})
	}
	return results, nil
}
