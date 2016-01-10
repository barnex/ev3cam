package main

import (
	"flag"
	"fmt"
	"image"
	"math"
	_ "net/http/pprof"
	"os"
)

var (
	flagDev     = flag.String("dev", "/dev/video0", "video device")
	flagFPS     = flag.Int("fps", 15, "maximum frames per second")
	flagHeight  = flag.Int("h", 240, "image height in pixels")
	flagPort    = flag.String("http", ":8080", "webserver port")
	flagQuality = flag.Int("quality", 50, "jpeg qualtity")
	flagWidth   = flag.Int("w", 320, "image width in pixels")
	flagV       = flag.Bool("v", true, "verbose output")
)

var (
	input  chan image.Image
	output = make(chan image.Image)
)

func main() {
	flag.Parse()

	pipe, err := openGStreamer()
	if err != nil {
		exit(err)
	}
	input = decodeMJPEG(pipe)
	output = runProcessing(input)

	if err := serveHTTP(); err != nil {
		exit(err)
	}
}

var (
	bg     [3]Floats
	filter = 0.75
)

func process(in [3]Floats) Floats {
	if bg[0] == nil {
		bg = in
	}

	out := makeFloats(in[0].Size())
	OUT := out.Data()
	IN := data(in)
	BG := data(bg)

	for i := range OUT {
		OUT[i] = math.Sqrt(sq(IN[0][i]-BG[0][i])+
			sq(IN[1][i]-BG[1][i])+
			sq(IN[2][i]-BG[2][i])) / math.Sqrt(3)
	}

	for c := range bg {
		for i := range BG[c] {
			BG[c][i] = (1-filter)*BG[c][i] + filter*IN[c][i]
		}
	}

	return out
}

func sq(x float64) float64 {
	return x * x
}
func data(x [3]Floats) [3][]float64 {
	return [3][]float64{x[0].Data(), x[1].Data(), x[2].Data()}
}

func runProcessing(input chan image.Image) chan image.Image {
	output := make(chan image.Image)
	go func() {
		for {
			img := <-input
			x := process(toVector(img))
			output <- Floats(x)
		}
	}()
	return output
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}
