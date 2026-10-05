//go:build ignore

// Command gen draws the app icon: go run desktop/build/icon.go out.png
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	bg, ring, dot := color.RGBA{7, 9, 13, 255}, color.RGBA{34, 42, 56, 255}, color.RGBA{74, 222, 128, 255}
	for y := range size {
		for x := range size {
			// A rounded square with a green dot, the "needs you" signal.
			if !inRoundedSquare(float64(x), float64(y), 100, 924, 190) {
				continue
			}
			d := math.Hypot(float64(x)-512, float64(y)-512)
			switch {
			case d < 170:
				img.Set(x, y, dot)
			case d < 215:
				img.Set(x, y, ring)
			default:
				img.Set(x, y, bg)
			}
		}
	}
	f, err := os.Create(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

func inRoundedSquare(x, y, lo, hi, r float64) bool {
	cx, cy := math.Min(math.Max(x, lo+r), hi-r), math.Min(math.Max(y, lo+r), hi-r)
	return math.Hypot(x-cx, y-cy) <= r
}
