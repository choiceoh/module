package ocr

import (
	"fmt"
	"image"

	"github.com/disintegration/imaging"
	ort "github.com/yalue/onnxruntime_go"
)

// recognizeText runs PP-OCRv4 recognition on a crop and returns the
// decoded string with an average confidence.
//
// Input shape: [1, 3, 48, W] where W = ceil(w/h * 48 / 32) * 32.
func recognizeText(crop image.Image) (string, float32, error) {
	b := crop.Bounds()
	if b.Dx() < 4 || b.Dy() < 4 {
		return "", 0, nil
	}

	const targetH = 48
	ratio := float64(b.Dx()) / float64(b.Dy())
	targetW := int(float64(targetH) * ratio)
	if targetW < 32 {
		targetW = 32
	}
	// Round up to multiple of 16 (PPOCR uses multiples of 8 but 16 is safer).
	if targetW%16 != 0 {
		targetW = (targetW/16 + 1) * 16
	}

	resized := imaging.Resize(crop, targetW, targetH, imaging.Linear)

	// Normalize: (pixel/255 - 0.5) / 0.5, giving range [-1, 1].
	mean := []float32{0.5, 0.5, 0.5}
	std := []float32{0.5, 0.5, 0.5}
	input := imageToCHW(resized, targetW, targetH, mean, std)

	inTensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(targetH), int64(targetW)), input)
	if err != nil {
		return "", 0, fmt.Errorf("rec input: %w", err)
	}
	defer inTensor.Destroy()

	// Rec output shape: [1, T, C] where C = len(dictChars) and T ≈ targetW/8.
	numClasses := len(dictChars)
	seqLen := int64(targetW / 8)
	outTensor, err := ort.NewEmptyTensor[float32](ort.NewShape(1, seqLen, int64(numClasses)))
	if err != nil {
		return "", 0, fmt.Errorf("rec output: %w", err)
	}
	defer outTensor.Destroy()

	session, err := ort.NewAdvancedSession(recPath,
		[]string{"x"},
		[]string{"softmax_11.tmp_0"},
		[]ort.Value{inTensor}, []ort.Value{outTensor}, nil)
	if err != nil {
		return "", 0, fmt.Errorf("rec session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return "", 0, fmt.Errorf("rec run: %w", err)
	}

	logits := outTensor.GetData()
	text, score := ctcDecode(logits, int(seqLen), numClasses)
	return text, score, nil
}

// imageToCHW converts an image to a flat CHW float32 tensor with
// channel-wise mean/std normalization (applied to 0..1 normalized pixels).
func imageToCHW(img image.Image, w, h int, mean, std []float32) []float32 {
	out := make([]float32, 3*w*h)
	offR := 0
	offG := w * h
	offB := 2 * w * h
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			fr := float32(r>>8) / 255.0
			fg := float32(g>>8) / 255.0
			fb := float32(b>>8) / 255.0
			out[offR+y*w+x] = (fr - mean[0]) / std[0]
			out[offG+y*w+x] = (fg - mean[1]) / std[1]
			out[offB+y*w+x] = (fb - mean[2]) / std[2]
		}
	}
	return out
}
