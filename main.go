package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	_ "net/http/pprof"
	"os"
	"strings"
)

var (
	flagSrc     = flag.String("src", "/dev/video0", "video device")
	flagRec     = flag.String("rec", "", "directory to record input files")
	flagFPS     = flag.Int("fps", 15, "maximum frames per second")
	flagHeight  = flag.Int("h", 480, "image height in pixels")
	flagPort    = flag.String("http", ":8080", "webserver port")
	flagQuality = flag.Int("quality", 50, "jpeg qualtity")
	flagWidth   = flag.Int("w", 640, "image width in pixels")
	flagV       = flag.Bool("v", true, "verbose output")
)

var (
	input                     chan image.Image
	output1                   = make(chan image.Image) // input with crosshairs on target
	output2                   = make(chan image.Image) // processed input with crosshairs
	targetX, targetY, targetM float64 // target coordinates and mass
)

func main() {
	flag.Parse()

	if strings.HasPrefix(*flagSrc, "/dev/") {
		pipe, err := openGStreamer()
		if err != nil {
			exit(err)
		}
		input = decodeMJPEG(pipe)
	} else {
		input = streamRecorded(*flagSrc)
	}

	runProcessing(input)

	for{
		record(<-output2)
	}

	if err := serveHTTP(); err != nil {
		exit(err)
	}
}

var (
	bg        [3]Floats // background image
	filter    = 0.75 // background image update rate
	threshold = 0.15 // motion threshold value (0..1)
)

// process updates targetX, targetY with the position the moving target.
func process(in [3]Floats) Floats {
	tProc.Start()
	if bg[0] == nil {
		bg = in
	}

	w, h := in[0].Size()
	out := makeFloats(w, h)
	OUT := out.Data()
	IN := data(in)
	BG := data(bg)

	for i := range OUT {
		diff := sqrt(sq(IN[0][i]-BG[0][i])+sq(IN[1][i]-BG[1][i])+sq(IN[2][i]-BG[2][i])) / sqrt(3)
		if diff > threshold {
			OUT[i] = 1
		}
	}

	for c := range bg {
		for i := range BG[c] {
			BG[c][i] = (1-filter)*BG[c][i] + filter*IN[c][i]
		}
	}

	var sX, sY, n float64
	for y := range out {
		for x := range out[y] {
			if out[y][x] == 1 {
				sX += float64(x)
				sY += float64(y)
				n++
			}
		}
	}
	if n > 0 {
		tX := (sX / n)
		tY := (sY / n)
		//m := float64(n)

		filterX := 1.0
		filterY := 0.3
		frac := float64(n) / float64(w*h)
		if frac < 0.003 {
			filterX = 0.1
			filterY = 0.1
		}

		targetX = (1-filterX)*targetX + filterX*tX
		targetY = (1-filterY)*targetY + filterY*tY
	}

	tProc.Stop()
	return out
}

// runProcessing reads images from input, determines the target
// position, and marks the target out the output channels.
func runProcessing(input chan image.Image) {
	go func() {
		for {
			img := <-input
			x := process(toVector(img))
			softPush(output1, mark(img))
			softPush(output2, mark(x))
		}
	}()
}

// softPush pushes img in ch, if possible. Never blocks.
func softPush(ch chan image.Image, img image.Image) {
	select {
	default:
	case ch <- img:
	}
}

// toVector turns an image into pixel values between 0 and 1.
// Employs a gamma of 2, close enough to sRGB to work well.
func toVector(im image.Image) [3]Floats {
	img := im.(*image.YCbCr)
	w := img.Bounds().Max.X
	h := img.Bounds().Max.Y
	f := makeVectors(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.YCbCrAt(x, y).RGBA()
			f[0][y][x] = sqrt(float64(r) / 0xffff)
			f[1][y][x] = sqrt(float64(g) / 0xffff)
			f[2][y][x] = sqrt(float64(b) / 0xffff)
		}
	}
	return f
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}

func sq(x float64) float64 {
	return x * x
}

func sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func data(x [3]Floats) [3][]float64 {
	return [3][]float64{x[0].Data(), x[1].Data(), x[2].Data()}
}
