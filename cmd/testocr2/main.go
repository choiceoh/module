package main
import (
	"fmt"; "os"; "time"
	"module-scanner/internal/ocr"
)
func main() {
	data, _ := os.ReadFile(os.Args[1])
	t := time.Now()
	r, err := ocr.RecognizeBytes(data)
	if err != nil { fmt.Println(err); return }
	raw := ocr.ExtractSerialCandidates(r, 6, 40)
	fmt.Printf("OCR %s: %d results, %d candidates\n", time.Since(t), len(r), len(raw))
	for _, s := range raw {
		fmt.Println(" ", s)
	}
}
