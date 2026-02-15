package main

import (
	"context"
	"flag"
	"os"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	catalogimporter "github.com/louisbranch/fracturing.space/internal/tools/importer/content/daggerheart/v1"
)

func main() {
	cfg, err := catalogimporter.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("Error: %v", err)
	}

	if err := catalogimporter.Run(context.Background(), cfg, os.Stdout); err != nil {
		config.Exitf("Error: %v", err)
	}
}
