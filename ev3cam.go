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
	"time"
)

var (
	flagPort = flag.String("http", ":8080", "webserver port")
	flagFPS  = flag.Int("fps", 10, "maximum frames per second")
)
var stream <-chan image.Image

var (
	start      time.Time
	nDropped   int
	nProcessed int
	nStreamed  int
	errors = make(map[string]int)
)

func main() {
	flag.Parse()

	in, err := openStream()
	if err != nil {
		exit(err)
	}

	http.Handle("/", appHandler(handleStatic))
	http.Handle("/img", appHandler(handleImg))
	http.Handle("/cam", appHandler(handleStream))
	stream = decodeStream(in)

	exec.Command("google-chrome", "http://localhost"+*flagPort).Start()

	if err := http.ListenAndServe(*flagPort, nil); err != nil {
		exit(err)
	}

}

func printStats() {
	if nStreamed == 0{
		return
	}
	if (start == time.Time{}){
		start = time.Now()
		return
	}
	fps := float64(nStreamed) / time.Since(start).Seconds()
		fmt.Printf("dropped:%v processed:%v streamed:%v fps:%.1f errors:%v\n", nDropped, nProcessed, nStreamed, fps, errors)
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

func handleImg(w http.ResponseWriter, r *http.Request) error {
	img := <-stream
	return jpeg.Encode(w, img, &jpeg.Options{Quality: 50})
}

func handleStream(w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "multipart/x-mixed-replace;boundary=BOUNDARY")
	for {
		img := <-stream
		fmt.Fprint(w, "--BOUNDARY\r\n"+
			"Content-Type: image/jpeg\r\n"+
			//"Content-Length:" + length + "\r\n" +
			"\r\n")

		err := jpeg.Encode(w, img, &jpeg.Options{Quality: 75})
		if err != nil {
			errors[err.Error()]++
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

func decodeStream(in io.Reader) <-chan image.Image {
	ch := make(chan image.Image)

	go func() {
		//in := bufio.NewReader(input)
		for {
			printStats()
			img, err := jpeg.Decode(in)
			if err != nil {
				if err.Error() == "unexpected EOF" {
					close(ch)
				}
				errors[err.Error()]++
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

func openStream() (io.Reader, error) {
	bin := "gst-launch-1.0"
	args := fmt.Sprintf(`v4l2src device=/dev/video0 ! videorate ! video/x-raw,framerate=%d/1 ! jpegenc ! filesink buffer-size=0 location=/dev/stdout`, *flagFPS)

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

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return stdout, nil
}
