package main

import (
	"fmt"
	"os"

	"github.com/louisbranch/fracturing.space/internal/tools/joingrant"
)

func main() {
	if err := joingrant.Run(os.Stdout, nil); err != nil {
		fmt.Fprintf(os.Stderr, "generate join grant key: %v\n", err)
		os.Exit(1)
	}
}
