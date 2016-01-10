package main

import (
	"flag"
	"fmt"
	"image"
	_ "net/http/pprof"
	"os"
	"path"
	"syscall"
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
	stream chan image.Image
	render = make(chan image.Image)
	fifo   = "fifo"
)

func main() {
	flag.Parse()

	fifo += path.Base(*flagDev)
	if err := syscall.Mkfifo(fifo, 0666); err != nil {
		fmt.Fprintln(os.Stderr, "mkfifo", fifo, ":", err)
	}

	in, err := openStream()
	if err != nil {
		exit(err)
	}

	stream = decodeStream(in)

	go runProcessing()

	//exec.Command("google-chrome", "http://localhost"+*flagPort).Start()

	if err := serveHTTP(); err != nil {
		exit(err)
	}
}

func runProcessing() {
	for {
		img := <-stream
		//fmt.Printf("%T %v", img, img.Bounds())

		f := toVector(img)[1]

		select {
		default:
			nDropped++
		case render <- Floats(f):
		}
	}
}

// sinkhole sucks the image stream so we can test intrinsic performance
func sinkhole() {
	go func() {
		for {
			<-stream
		}
	}()
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}
