package main

import (
	"bufio"
	"fmt"
	"image/jpeg"
	"os"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	for {
		img, err := jpeg.Decode(in)
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(img.Bounds())
	}
}
