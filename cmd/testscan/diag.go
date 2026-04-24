//go:build ignore

package main

// Separate diagnostic: crop the middle column only, try decode.
// Run with:  go run cmd/testscan/diag.go <image>
import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"github.com/disintegration/imaging"

	"module-scanner/internal/barcode"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: diag <image>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	fmt.Printf("image: %dx%d\n", w, h)

	// Extract the middle SN column: roughly X = w * 0.35 .. 0.72
	mid := imaging.Crop(img, image.Rect(w*35/100, 0, w*72/100, h))
	fmt.Printf("middle crop: %dx%d\n", mid.Bounds().Dx(), mid.Bounds().Dy())

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, mid, imaging.PNG); err != nil {
		panic(err)
	}
	hits, err := barcode.DecodeAll(buf.Bytes())
	if err != nil {
		fmt.Printf("mid decode error: %v\n", err)
		return
	}
	fmt.Printf("mid decoded: %d\n", len(hits))
	for i, h := range hits {
		fmt.Printf("  %d: %s\n", i+1, h.Text)
	}
}
