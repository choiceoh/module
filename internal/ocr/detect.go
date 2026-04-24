package ocr

import (
	"fmt"
	"image"
	"image/color"

	"github.com/disintegration/imaging"
	ort "github.com/yalue/onnxruntime_go"
)

// Box is an axis-aligned rectangle in image coordinates.
type Box struct {
	X0, Y0, X1, Y1 int
	Score          float32
}

// detectTextBoxes runs PP-OCRv4 detection, thresholds the probability
// map, extracts connected components, and returns padded bounding boxes
// for each detected text region.
func detectTextBoxes(img image.Image) ([]Box, error) {
	// Resize so long side is 960 while staying a multiple of 32.
	// This is what RapidOCR uses for detection.
	const maxSide = 960
	orig := img.Bounds()
	ow, oh := orig.Dx(), orig.Dy()

	ratio := 1.0
	if ow > oh {
		if ow > maxSide {
			ratio = float64(maxSide) / float64(ow)
		}
	} else {
		if oh > maxSide {
			ratio = float64(maxSide) / float64(oh)
		}
	}
	rw := int(float64(ow) * ratio)
	rh := int(float64(oh) * ratio)
	rw = (rw/32 + 1) * 32
	rh = (rh/32 + 1) * 32
	if rw > maxSide {
		rw = maxSide
	}
	if rh > maxSide*2 {
		rh = maxSide * 2
	}

	resized := imaging.Resize(img, rw, rh, imaging.Linear)

	// Convert to CHW float32, normalize with ImageNet-style mean/std
	// that PPOCR detection uses: mean=[0.485,0.456,0.406] std=[0.229,0.224,0.225]
	input := imageToCHW(resized, rw, rh, []float32{0.485, 0.456, 0.406}, []float32{0.229, 0.224, 0.225})

	inTensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(rh), int64(rw)), input)
	if err != nil {
		return nil, fmt.Errorf("det input: %w", err)
	}
	defer inTensor.Destroy()

	// Pre-declare empty output tensor of known dtype; advanced API will size it.
	outTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 1, int64(rh), int64(rw)))
	if err != nil {
		return nil, fmt.Errorf("det output: %w", err)
	}
	defer outTensor.Destroy()

	session, err := ort.NewAdvancedSession(detPath,
		[]string{"x"},
		[]string{"sigmoid_0.tmp_0"},
		[]ort.Value{inTensor}, []ort.Value{outTensor}, nil)
	if err != nil {
		return nil, fmt.Errorf("det session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("det run: %w", err)
	}

	prob := outTensor.GetData() // [1, 1, rh, rw]

	// Convert prob map to binary mask at threshold 0.3.
	mask := image.NewGray(image.Rect(0, 0, rw, rh))
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			if prob[y*rw+x] > 0.3 {
				mask.SetGray(x, y, color.Gray{255})
			}
		}
	}

	// Extract connected components and return bounding boxes mapped
	// back to the original image coordinate space, expanded outward.
	regions := connectedComponents(mask)
	boxes := make([]Box, 0, len(regions))
	for _, r := range regions {
		if r.w < 6 || r.h < 4 {
			continue
		}
		// Expand 1.5x in each dimension (simpler than full DB unshrink).
		padX := r.w / 3
		padY := r.h
		x0 := r.x - padX
		y0 := r.y - padY/2
		x1 := r.x + r.w + padX
		y1 := r.y + r.h + padY/2
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > rw {
			x1 = rw
		}
		if y1 > rh {
			y1 = rh
		}

		// Map back to original coords.
		sx := float64(ow) / float64(rw)
		sy := float64(oh) / float64(rh)
		boxes = append(boxes, Box{
			X0:    int(float64(x0) * sx),
			Y0:    int(float64(y0) * sy),
			X1:    int(float64(x1) * sx),
			Y1:    int(float64(y1) * sy),
			Score: float32(r.score),
		})
	}
	return boxes, nil
}
