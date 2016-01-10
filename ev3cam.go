package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"log"
	"net/http"
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
	flagQuality = flag.Int("quality", 25, "jpeg qualtity")
	flagWidth   = flag.Int("w", 320, "image width in pixels")
	flagV       = flag.Bool("v", true, "verbose output")
)

var (
	stream = make(chan image.Image)
	render = make(chan image.Image)
	fifo   = "fifo"
)

// for performance statistics
var (
	start      time.Time
	count      int // don't print every time
	nBytes     int64
	nDropped   int
	nProcessed int
	nStreamed  int
	nErrors    int
	tEnc, tDec timer
)


func main() { Main() }

func Main() {
	flag.Parse()

	fifo += path.Base(*flagDev)
	if err := syscall.Mkfifo(fifo, 0666); err != nil {
		fmt.Fprintln(os.Stderr, "mkfifo", fifo, ":", err)
	}

	in, err := openStream()
	if err != nil {
		exit(err)
	}

	http.Handle("/", appHandler(handleStatic))
	http.Handle("/cam", mjpegHandler(stream))
	http.Handle("/processed", mjpegHandler(render))

	decodeStream(in)

	go runProcessing()

	//exec.Command("google-chrome", "http://localhost"+*flagPort).Start()

	if err := http.ListenAndServe(*flagPort, nil); err != nil {
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

type Floats [][]float64

func (f Floats) At(x, y int) color.Color {
	return color.Gray16{uint16(f[y][x] * 0xffff)}
}

func (f Floats) ColorModel() color.Model {
	return color.Gray16Model //??
}

func (f Floats) Bounds() image.Rectangle {
	h := len(f)
	w := len(f[0])
	return image.Rect(0, 0, w, h)
}

// TODO: srgb?
func toVector(im image.Image) [3][][]float64 {
	img := im.(*image.YCbCr)
	w := img.Bounds().Max.X
	h := img.Bounds().Max.Y
	f := makeVectors(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.YCbCrAt(x, y).RGBA()
			f[0][y][x] = float64(r) / 0xffff
			f[1][y][x] = float64(g) / 0xffff
			f[2][y][x] = float64(b) / 0xffff
		}
	}
	return f
}

func makeScalars(w, h int) [][]float64 {
	storage := make([]float64, w*h)
	s := make([][]float64, h)
	for y := range s {
		s[y] = storage[y*w : (y+1)*w]
	}
	return s
}

func makeVectors(w, h int) [3][][]float64 {
	var v [3][][]float64
	for c := 0; c < 3; c++ {
		v[c] = makeScalars(w, h)
	}
	return v
}

// sinkhole sucks the image stream so we can test intrinsic performance
func sinkhole() {
	go func() {
		for {
			<-stream
		}
	}()

}

func printStats() {
	if !*flagV {
		return
	}
	count++
	if count%16 != 0 {
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

	fmt.Printf("%.1fkB/s, decode:%.1f/s drop:%.1f/s render:%.1f errors/s:%.1f\n", kBps, pps, dps, fps, eps)
	fmt.Println("decode", &tDec, "encode", &tEnc)
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


type mjpegHandler chan image.Image

func(h mjpegHandler)ServeHTTP(w http.ResponseWriter, r*http.Request){
	w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary=--BOUNDARY")
	for {
		img := <-h
		_, err := fmt.Fprint(w, "--BOUNDARY\r\n"+
			"Content-Type: image/jpeg\r\n"+
			//"Content-Length:" + length + "\r\n" +
			"\r\n")
		if err != nil {
			nErrors++
			http.Error(w, err.Error(), 500)
			return
		}

		tEnc.Start()
		err = jpeg.Encode(w, img, &jpeg.Options{Quality: *flagQuality})
		tEnc.Stop()
		if err != nil {
			nErrors++
			http.Error(w, err.Error(), 500)
			return
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
