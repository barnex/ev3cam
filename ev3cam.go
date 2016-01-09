package main

import (
	"bufio"
	"fmt"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"strings"
)

func main() {
	in, err := openStream()
	if err != nil {
		exit(err)
	}
	decodeStream(in)
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

func decodeStream(input io.Reader) {
	in := bufio.NewReader(input)
	for {
		img, err := jpeg.Decode(in)
		if err != nil {
			fmt.Println(err)
			if err.Error() == "unexpected EOF" {
				exit()
			}
			continue
		}
		fmt.Println(img.Bounds())
	}
}

func exit(x ...interface{}) {
	fmt.Fprintln(os.Stderr, x...)
	os.Exit(1)
}
