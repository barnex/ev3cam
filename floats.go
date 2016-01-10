package main

import (
	"image"
	"image/color"
)

type Floats [][]float64

func (f Floats) At(x, y int) color.Color {
	return color.Gray16{uint16(f[y][x] * 0xffff)}
}

func (f Floats) ColorModel() color.Model {
	return color.Gray16Model //??
}

func (f Floats) Size() (w, h int) {
	h = len(f)
	w = len(f[0])
	return
}

func (f Floats) Len() int {
	w, h := f.Size()
	return w * h
}

func (f Floats) Bounds() image.Rectangle {
	w, h := f.Size()
	return image.Rect(0, 0, w, h)
}

func (f Floats) Data() []float64 {
	return f[0][:f.Len()]
}

func makeFloats(w, h int) Floats {
	storage := make([]float64, w*h)
	s := make([][]float64, h)
	for y := range s {
		s[y] = storage[y*w : (y+1)*w]
	}
	return s
}

// TODO: srgb?
func toVector(im image.Image) [3]Floats {
	img := im.(*image.YCbCr)
	w := img.Bounds().Max.X
	h := img.Bounds().Max.Y
	f := makeVectors(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.YCbCrAt(x, y).RGBA()
			f[0][y][x] = float64(r) / 0xffff
			f[1][y][x] = float64(g) / 0xffff
			f[2][y][x] = float64(b) / 0xffff
		}
	}
	return f
}

func makeVectors(w, h int) [3]Floats {
	var v [3]Floats
	for c := 0; c < 3; c++ {
		v[c] = makeFloats(w, h)
	}
	return v
}
