package main

import (
	"flag"
	"fmt"
	"image"
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

func runProcessing(input chan image.Image) chan image.Image {
	output := make(chan image.Image)
	go func() {
		for {
			img := <-input
			f := toVector(img)[1]
			output <- Floats(f)
		}
	}()
	return output
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}
