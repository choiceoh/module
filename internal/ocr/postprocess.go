package ocr

import (
	"image"
)

type region struct {
	x, y, w, h int
	score      float64
}

// connectedComponents finds 4-connected regions of non-zero pixels in a
// binary mask and returns each region's bounding rectangle.
//
// Iterative flood fill via a pixel-index queue; works for 2000×3000
// masks in a few ms.
func connectedComponents(mask *image.Gray) []region {
	w := mask.Rect.Dx()
	h := mask.Rect.Dy()
	visited := make([]bool, w*h)

	var regions []region
	queue := make([]int, 0, 1024)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			if visited[idx] || mask.Pix[idx] == 0 {
				continue
			}
			// BFS
			queue = queue[:0]
			queue = append(queue, idx)
			visited[idx] = true

			minX, minY, maxX, maxY := x, y, x, y
			count := 0
			for qi := 0; qi < len(queue); qi++ {
				p := queue[qi]
				count++
				px := p % w
				py := p / w
				if px < minX {
					minX = px
				}
				if px > maxX {
					maxX = px
				}
				if py < minY {
					minY = py
				}
				if py > maxY {
					maxY = py
				}
				if px > 0 {
					n := p - 1
					if !visited[n] && mask.Pix[n] != 0 {
						visited[n] = true
						queue = append(queue, n)
					}
				}
				if px < w-1 {
					n := p + 1
					if !visited[n] && mask.Pix[n] != 0 {
						visited[n] = true
						queue = append(queue, n)
					}
				}
				if py > 0 {
					n := p - w
					if !visited[n] && mask.Pix[n] != 0 {
						visited[n] = true
						queue = append(queue, n)
					}
				}
				if py < h-1 {
					n := p + w
					if !visited[n] && mask.Pix[n] != 0 {
						visited[n] = true
						queue = append(queue, n)
					}
				}
			}
			regions = append(regions, region{
				x: minX, y: minY,
				w: maxX - minX + 1, h: maxY - minY + 1,
				score: float64(count),
			})
		}
	}
	return regions
}

// ctcDecode implements greedy CTC decoding for PP-OCR recognition
// output of shape [T, C] (single sequence). Returns the decoded text
// and mean confidence score.
func ctcDecode(logits []float32, seqLen, numClasses int) (string, float32) {
	out := ""
	var totalScore float32
	validCount := 0
	prev := -1
	for t := 0; t < seqLen; t++ {
		base := t * numClasses
		// argmax
		best := 0
		bestVal := logits[base]
		for c := 1; c < numClasses; c++ {
			if logits[base+c] > bestVal {
				bestVal = logits[base+c]
				best = c
			}
		}
		// collapse repeats, drop blanks (index 0)
		if best != prev && best != 0 {
			if best < len(dictChars) {
				out += dictChars[best]
				totalScore += bestVal
				validCount++
			}
		}
		prev = best
	}
	score := float32(1.0)
	if validCount > 0 {
		score = totalScore / float32(validCount)
	}
	return out, score
}
