// Package main runs one live AI GM evaluation lane and prints Promptfoo-facing JSON.
package main

import (
	"context"
	"flag"
	"os"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/cli"

	aievalcmd "github.com/louisbranch/fracturing.space/internal/cmd/aieval"
)

func main() {
	cfg, err := aievalcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("Error: %v", err)
	}

	ctx, stop := cli.WithSignalContext(context.Background())
	defer stop()

	if err := aievalcmd.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		config.Exitf("Error: %v", err)
	}
}
