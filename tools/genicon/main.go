// genicon menulis assets/app_icon.png (ikon aplikasi untuk winres / taskbar Windows).
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
)

func main() {
	const n = 256
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	bg := color.NRGBA{R: 13, G: 148, B: 136, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)

	cx, cy := n / 2, n / 2
	r := n * 22 / 100
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}

	// Jalankan dari root modul: go run ./tools/genicon
	out := "assets/app_icon.jpg"
	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
