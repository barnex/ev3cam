package main

import (
	"bufio"
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
)

var (
	flagPort = flag.String("http", ":8080", "webserver port")
)

var (
	stream    <-chan image.Image
	nDropped  int
	nPiped    int
	nStreamed int
)

func main() {
	in, err := openStream()
	if err != nil {
		exit(err)
	}

	http.Handle("/", appHandler(handleStatic))
	http.Handle("/img", appHandler(handleImg))
	http.Handle("/cam", appHandler(handleStream))
	stream = decodeStream(in)

	if err := http.ListenAndServe(*flagPort, nil); err != nil {
		exit(err)
	}
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
			"Content-Type:image/jpeg\r\n"+
			//"Content-Length:" + length + "\r\n" +
			"\r\n")

		err := jpeg.Encode(w, img, &jpeg.Options{Quality: 75})
		if err != nil {
			fmt.Println(err)
		}
		nStreamed++
		fmt.Println("streamed", nStreamed, "frames")
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
		in := bufio.NewReader(input)
		for {
			img, err := jpeg.Decode(in)
			if err != nil {
				if err.Error() == "unexpected EOF" {
					close(ch)
				}
				fmt.Println(err)
				continue
			}
			fmt.Println(img.Bounds)
			select {
			default:
				nDropped++
				fmt.Println("dropped", nDropped, "frames")
			case ch <- img:
				nPiped++
				fmt.Println("piped", nPiped, "frames")
			}
		}
	}()
	return ch
}

func openStream() (io.Reader, error) {
	bin := "gst-launch-1.0"
	args := strings.Split(`v4l2src device=/dev/video0 ! videorate ! video/x-raw,framerate=3/1 ! jpegenc ! filesink buffer-size=1 location=/dev/stdout`, " ")

	fmt.Println(bin, args)
	cmd := exec.Command(bin, args...)

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
