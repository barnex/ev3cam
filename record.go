package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"os"
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

func streamRecorded(dir string) {
	go func() {
		n := 0
		for {
			f, err := os.Open(fmt.Sprintf("%v/%06d.jpg", *flagSrc, n))
			if err != nil {
				exit(err)
			}
			img, err := jpeg.Decode(f)
			if err != nil {
				exit(err)
			}
			f.Close()
			input <- img
			n++
		}
	}()
}
