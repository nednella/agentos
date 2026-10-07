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

type rgb struct{ r, g, b float64 }

var (
	bgTop    = rgb{25, 29, 37}
	bgBottom = rgb{13, 15, 19}
	text     = rgb{230, 233, 239}
	accent   = rgb{167, 139, 250}
	white    = rgb{255, 255, 255}
)

// A terminal prompt: a chevron and a cursor in the app's accent.
func main() {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			px, py := float64(x)+0.5, float64(y)+0.5
			square := roundedRect(px, py, 100, 100, 924, 924, 185)
			alpha := coverage(square)
			if alpha == 0 {
				continue
			}
			c := mix(bgTop, bgBottom, (py-100)/824)
			c = mix(c, white, 0.07*coverage(math.Abs(square+2)-2))
			cursor := roundedRect(px, py, 520, 604, 740, 676, 20)
			// The edge of a Gaussian-blurred rectangle, as the cursor's glow.
			if cursor > 0 {
				c = mix(c, accent, 0.35*math.Erfc(cursor/(26*math.Sqrt2)))
			}
			c = mix(c, accent, coverage(cursor))
			chevron := math.Min(segment(px, py, 290, 380, 450, 512), segment(px, py, 450, 512, 290, 644)) - 32
			c = mix(c, text, coverage(chevron))
			img.Set(x, y, color.NRGBA{uint8(c.r + 0.5), uint8(c.g + 0.5), uint8(c.b + 0.5), uint8(255*alpha + 0.5)})
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

// coverage turns a signed distance into pixel coverage, which anti-aliases every edge.
func coverage(d float64) float64 { return math.Max(0, math.Min(1, 0.5-d)) }

func mix(a, b rgb, t float64) rgb {
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// roundedRect is the signed distance from (x, y) to the rectangle (x0, y0)-(x1, y1) with corner radius r.
func roundedRect(x, y, x0, y0, x1, y1, r float64) float64 {
	cx, cy := math.Min(math.Max(x, x0+r), x1-r), math.Min(math.Max(y, y0+r), y1-r)
	inside := math.Max(math.Max(x0+r-x, x-(x1-r)), math.Max(y0+r-y, y-(y1-r)))
	if inside < 0 {
		return inside - r
	}
	return math.Hypot(x-cx, y-cy) - r
}

// segment is the distance from (x, y) to the segment (ax, ay)-(bx, by).
func segment(x, y, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	t := math.Max(0, math.Min(1, ((x-ax)*dx+(y-ay)*dy)/(dx*dx+dy*dy)))
	return math.Hypot(x-ax-t*dx, y-ay-t*dy)
}
