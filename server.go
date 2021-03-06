package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
)

func serveHTTP() error {
	http.Handle("/", appHandler(handleStatic))
	http.Handle("/input", mjpegHandler(input))
	http.Handle("/output1", mjpegHandler(output1))
	http.Handle("/output2", mjpegHandler(output2))
	return http.ListenAndServe(*flagPort, nil)
}

// mjpegHandler streams images as MJPEG.
type mjpegHandler chan image.Image

func (h mjpegHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func handleStatic(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintln(w, `
		<html>
		<head>
		</head>
		<body>
		<img src="/output1"></img>
		<img src="/output2"></img>
		</body>
		</html>
	`)
	return nil
}
