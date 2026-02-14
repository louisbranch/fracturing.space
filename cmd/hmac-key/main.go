package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/louisbranch/fracturing.space/internal/tools/hmackey"
)

func main() {
	cfg, err := hmackey.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse flags: %v\n", err)
		os.Exit(1)
	}
	if err := hmackey.Run(cfg, os.Stdout, nil); err != nil {
		fmt.Fprintf(os.Stderr, "generate key: %v\n", err)
		os.Exit(1)
	}
}
