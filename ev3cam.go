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
	stream   <-chan image.Image
	nDropped int
	nPiped   int
)

func main() {
	in, err := openStream()
	if err != nil {
		exit(err)
	}

	http.Handle("/img", appHandler(handleImg))
	stream = decodeStream(in)

	if err := http.ListenAndServe(*flagPort, nil); err != nil {
		exit(err)
	}
}

func handleImg(w http.ResponseWriter, r *http.Request) error {
	img := <-stream
	return jpeg.Encode(w, img, &jpeg.Options{Quality: 75})
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
			select {
			default:
				nDropped++
				fmt.Println("dropped", nDropped, "frames")
			case ch <- img:
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
