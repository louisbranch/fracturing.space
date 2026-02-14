// Package main provides a CLI for running Lua scenario scripts.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/platform/config"

	scenariocmd "github.com/louisbranch/fracturing.space/internal/cmd/scenario"
)

func main() {
	cfg, err := scenariocmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("Error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := scenariocmd.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		config.Exitf("Error: %v", err)
	}
}
