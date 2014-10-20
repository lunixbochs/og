package main

import (
	"os"
)

func main() {
	og := NewOg(os.Args[1:], ".")
	if len(os.Args) < 2 {
		og.Usage()
		return
	}
}
