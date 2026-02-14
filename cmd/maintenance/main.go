// Package main provides maintenance utilities.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/platform/config"
	"github.com/louisbranch/fracturing.space/internal/tools/maintenance"
)

func main() {
	cfg, err := maintenance.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("Error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if err := maintenance.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		config.Exitf("Error: %v", err)
	}
}
