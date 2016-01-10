package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
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
	stream = make(chan image.Image)
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

	decodeStream(in)

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

func decodeStream(input io.Reader) {
	go func() {
		in := newReader(bufio.NewReaderSize(input, 64*1024*1024))
		//in := newReader(input)
		for {
			printStats()
			tDec.Start()
			img, err := jpeg.Decode(in)
			tDec.Stop()
			if err != nil {
				if err.Error() == "unexpected EOF" {
					close(stream)
				}
				if err.Error() != "invalid JPEG format: missing SOI marker" {
					exit(err)
				}
				nErrors++
				continue
			}
			select {
			default:
				nDropped++
			case stream <- img:
				nProcessed++
			}
		}
	}()
}

type reader struct {
	in io.Reader
}

func newReader(in io.Reader) *reader {
	return &reader{in: in}
}

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.in.Read(p)
	nBytes += int64(n)
	return
}

func openStream() (io.Reader, error) {
	bin := "gst-launch-1.0"
	args := fmt.Sprintf(`v4l2src device=%s ! video/x-raw,framerate=%d/1,width=%d,height=%d ! jpegenc quality=%d ! filesink buffer-size=0 location=%v`, *flagDev, *flagFPS, *flagWidth, *flagHeight, *flagQuality, fifo)

	fmt.Println(bin, args)
	cmd := exec.Command(bin, strings.Split(args, " ")...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	go io.Copy(os.Stderr, stderr)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	go io.Copy(os.Stderr, stdout)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	f, err := os.Open(fifo)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type timer struct {
	n       int64
	total   time.Duration
	started time.Time
}

func (t *timer) Start() {
	t.started = time.Now()
}

func (t *timer) Stop() {
	t.n++
	t.total += time.Since(t.started)
	t.started = time.Time{}
}

func (t *timer) String() string {
	if t.n == 0 {
		return "0"
	}
	return time.Duration(int64(t.total) / t.n).String()
}
