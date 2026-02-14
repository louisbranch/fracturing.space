// Package main provides a CLI for seeding the local development database
// with demo data by exercising the full MCP->game stack, or by generating
// dynamic scenarios directly via gRPC.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/platform/config"

	seedcmd "github.com/louisbranch/fracturing.space/internal/cmd/seed"
)

func main() {
	cfg, err := seedcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		config.Exitf("Error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if err := seedcmd.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		config.Exitf("Error: %v", err)
	}
}
