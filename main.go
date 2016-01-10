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
	fifo   = "fifo"
)

func main() {
	flag.Parse()

	pipe, err := openGStreamer()
	if err != nil {
		exit(err)
	}
	input = decodeMJPEG(pipe)
	output = runProcessing(input)

	//exec.Command("google-chrome", "http://localhost"+*flagPort).Start()

	if err := serveHTTP(); err != nil {
		exit(err)
	}
}

func runProcessing(input chan image.Image) chan image.Image {
	output := make(chan image.Image)
	go func() {
		for {
			img := <-input
			//fmt.Printf("%T %v", img, img.Bounds())

			f := toVector(img)[1]

			select {
			default:
				nDropped++
			case output <- Floats(f):
			}
		}
	}()
	return output
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}
