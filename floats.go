package main

import (
	"image"
	"image/color"
)

// Floats turns a matrix into a grayscale image.
type Floats [][]float64

func (f Floats) At(x, y int) color.Color {
	return color.Gray16{uint16(clamp(sq(f[y][x])) * 0xffff)}
}

func (f Floats) ColorModel() color.Model {
	return color.Gray16Model
}

func (f Floats) Bounds() image.Rectangle {
	w, h := f.Size()
	return image.Rect(0, 0, w, h)
}

// Size is shorthand for getting the image bounds.
func (f Floats) Size() (w, h int) {
	h = len(f)
	w = len(f[0])
	return
}

// Len returns the total number of pixels.
func (f Floats) Len() int {
	w, h := f.Size()
	return w * h
}


// Data returns the pixel data as a contiguous list.
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


// clamp limits x to at most 1.
func clamp(x float64) float64 {
	if x > 1 {
		return 1
	}
	return x
}

