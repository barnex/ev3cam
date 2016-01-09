package main

import "testing"

func init() {
	go Main()
}

func BenchmarkAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		<-stream
	}
}
