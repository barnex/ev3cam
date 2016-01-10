package main

import (
	"fmt"
	"time"
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
