package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"os"
	"time"
)

var rec chan image.Image

func record(img image.Image) {
	if rec == nil {
		rec = make(chan image.Image, 10)
		os.Mkdir(*flagRec, 0777)
		go func() {
			n := 0
			for {
				f, err := os.Create(fmt.Sprintf("%v/%06d.jpg", *flagRec, n))
				if err != nil {
					exit(err)
				}
				img := <-rec
				b := bufio.NewWriter(f)
				if err := jpeg.Encode(b, img, &jpeg.Options{Quality: *flagQuality}); err != nil {
					exit(err)
				}
				b.Flush()
				f.Close()
				n++
			}
		}()
	}
	rec <- img
}

func streamRecorded(dir string) chan image.Image {
	input := make(chan image.Image)
	go func() {
		n := 0
		for {
			f, err := os.Open(fmt.Sprintf("%v/%06d.jpg", *flagSrc, n))
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Second)
				n = 0
				continue
			}
			img, err := jpeg.Decode(f)
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Second)
				n = 0
				continue
			}
			f.Close()
			input <- img
			time.Sleep(80 * time.Millisecond)
			n++
		}
	}()
	return input
}

func mark(src image.Image) image.Image {

	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)

	w, h := b.Max.X, b.Max.Y
	red := color.RGBA{255, 0, 0, 255}
	targetX := int(targetX)
	targetY := int(targetY)
	if targetX < 0 || targetX >= w || targetY < 0 || targetY >= h {
		fmt.Println("invalid target", targetX, targetY)
	} else {
		y := targetY
		for x := 0; x < w; x++ {
			dst.Set(x, y, red)
		}
		x := targetX
		for y := 0; y < h; y++ {
			dst.Set(x, y, red)
		}
	}
	return dst
}
