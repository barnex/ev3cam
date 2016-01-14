package main

import (
	"bufio"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

func openGStreamer() (io.Reader, error) {
	fifo := "fifo" + path.Base(*flagSrc)
	if err := syscall.Mkfifo(fifo, 0666); err != nil {
		fmt.Fprintln(os.Stderr, "mkfifo", fifo, ":", err)
	}

	bin := "gst-launch-1.0"
	args := fmt.Sprintf(`v4l2src device=%s ! video/x-raw,framerate=%d/1,width=%d,height=%d ! jpegenc quality=%d ! filesink buffer-size=0 location=%v`, *flagSrc, *flagFPS, *flagWidth, *flagHeight, *flagQuality, fifo)

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

func decodeMJPEG(input io.Reader) chan image.Image {
	stream := make(chan image.Image)
	go func() {
		in := newReader(bufio.NewReaderSize(input, 64*1024*1024))
		//in := newReader(input)
		for {
			printStats()
			tDec.Start()
			img, err := jpeg.Decode(in)
			tDec.Stop()
			//if *flagRec != "" {
			//	record(img)
			//}
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
	return stream
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
