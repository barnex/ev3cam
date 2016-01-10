package main

import (
	"fmt"
	"time"
)

// for performance statistics
var (
	start             time.Time
	count             int // don't print every time
	nBytes            int64
	nDropped          int
	nProcessed        int
	nStreamed         int
	nErrors           int
	tEnc, tDec, tProc timer
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
	fmt.Println("decode", &tDec, "process", &tProc, "encode", &tEnc)
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
