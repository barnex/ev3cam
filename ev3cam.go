package main

import (
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var (
	flagPort = flag.String("http", ":8080", "webserver port")
	flagFPS  = flag.Int("fps", 10, "maximum frames per second")
)

var (
	stream <-chan image.Image
	fifo   = "fifo"
)

var (
	start      time.Time
	nBytes     int64
	nDropped   int
	nProcessed int
	nStreamed  int
	nErrors    int
)

func main() {
	flag.Parse()

	if err := syscall.Mkfifo(fifo, 0666); err != nil {
		fmt.Fprintln(os.Stderr, "mkfifo", fifo, ":", err)
	}

	in, err := openStream()
	if err != nil {
		exit(err)
	}

	http.Handle("/", appHandler(handleStatic))
	http.Handle("/cam", appHandler(handleStream))
	stream = decodeStream(in)

	exec.Command("google-chrome", "http://localhost"+*flagPort).Start()

	if err := http.ListenAndServe(*flagPort, nil); err != nil {
		exit(err)
	}

}

var count = 0

func printStats() {
	count++
	if count%8 != 0 {
		return
	}
	now := time.Now()
	s := now.Sub(start).Seconds()
	start = now

	fps := float64(nStreamed) / s
	nStreamed = 0
	kBps := (float64(nBytes) / 1000) / s
	nBytes = 0
	eps := (float64(nErrors)) / s
	nErrors = 0
	dps := (float64(nDropped)) / s
	nDropped = 0
	pps := (float64(nProcessed)) / s
	nProcessed = 0

	fmt.Printf("%.1fkB/s, dropped:%.1f/s processed:%.1f/s fps:%.1f errors/s:%.1f\n", kBps, dps, pps, fps, eps)
}

func handleStatic(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintln(w, `
		<html>
		<head>
		</head>
		<body>
		<img src="/cam"></img>
		</body>
		</html>
	`)
	return nil
}

func handleStream(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary=BOUNDARY")
	for {
		img := <-stream
		fmt.Fprint(w, "--BOUNDARY\r\n"+
			"Content-Type: image/jpeg\r\n"+
			//"Content-Length:" + length + "\r\n" +
			"\r\n")

		err := jpeg.Encode(w, img, &jpeg.Options{Quality: 25})
		if err != nil {
			nErrors++
		}
		nStreamed++
	}
}

type appHandler func(w http.ResponseWriter, r *http.Request) error

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
	}
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}

func decodeStream(input io.Reader) <-chan image.Image {
	ch := make(chan image.Image)

	go func() {
		//in := newReader(bufio.NewReaderSize(input, 64*1024*1024))
		in := newReader(input)
		for {
			printStats()
			img, err := jpeg.Decode(in)
			if err != nil {
				if err.Error() == "unexpected EOF" {
					close(ch)
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
			case ch <- img:
				nProcessed++
			}
		}
	}()
	return ch
}

type reader struct {
	in io.Reader
}

func newReader(in io.Reader) *reader {
	return &reader{in: in}
}

func (r *reader) Read(p []byte) (n int, err error) {
	n, err = r.in.Read(p)
	//fmt.Println("read", len(p), "returns", n)
	nBytes += int64(n)
	return
}

func openStream() (io.Reader, error) {
	bin := "gst-launch-1.0"
	args := fmt.Sprintf(`v4l2src device=/dev/video0 ! videorate ! video/x-raw,framerate=%d/1 ! jpegenc ! filesink buffer-size=0 location=%v`, *flagFPS, fifo)

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
