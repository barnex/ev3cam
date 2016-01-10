package main

import (
	"image"
	"image/color"
)

type Floats [][]float64

func (f Floats) At(x, y int) color.Color {
	return color.Gray16{uint16(clamp(sq(f[y][x])) * 0xffff)}
}

func clamp(x float64) float64 {
	if x > 1 {
		return 1
	}
	return x
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

func makeVectors(w, h int) [3]Floats {
	var v [3]Floats
	for c := 0; c < 3; c++ {
		v[c] = makeFloats(w, h)
	}
	return v
}
