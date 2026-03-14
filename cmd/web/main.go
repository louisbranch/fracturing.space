// Package main starts the browser-facing web service.
//
// This process owns route wiring and static template serving so campaign/auth
// context is translated consistently for browsers and web users.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	webcmd "github.com/louisbranch/fracturing.space/internal/cmd/web"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil)).With("service", "web")
	slog.SetDefault(logger)

	cfg, err := webcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		logger.Error("parse web config", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := webcmd.Run(ctx, cfg); err != nil {
		logger.Error("serve web", "error", err)
		os.Exit(1)
	}
}
