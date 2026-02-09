package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

func main() {
	var bytes int
	flag.IntVar(&bytes, "bytes", 32, "number of random bytes (default: 32)")
	flag.Parse()

	if bytes <= 0 {
		fmt.Fprintln(os.Stderr, "bytes must be greater than zero")
		os.Exit(1)
	}

	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		fmt.Fprintf(os.Stderr, "generate random bytes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(hex.EncodeToString(buf))
}
