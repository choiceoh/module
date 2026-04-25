package main

import (
	"fmt"
	"os"
	"time"

	"module-scanner/internal/ocr"
)

func main() {
	t0 := time.Now()
	if err := ocr.Init(); err != nil {
		fmt.Println("init:", err)
		return
	}
	fmt.Printf("init: %s\n", time.Since(t0).Round(time.Millisecond))
	defer ocr.Cleanup()
	for i, p := range os.Args[1:] {
		t := time.Now()
		r, err := ocr.Recognize(p)
		fmt.Printf("#%d %d results in %s err=%v\n", i+1, len(r), time.Since(t).Round(time.Millisecond), err)
	}
	fmt.Printf("total: %s\n", time.Since(t0).Round(time.Millisecond))
}
