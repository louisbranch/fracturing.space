package main

import (
	"os"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/joingrant"
)

func main() {
	if err := joingrant.Run(os.Stdout, nil); err != nil {
		config.Exitf("generate join grant key: %v", err)
	}
}
