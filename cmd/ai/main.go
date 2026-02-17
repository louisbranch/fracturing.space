// Package main wires the AI gRPC service process lifecycle.
//
// It reads config from flags/env and runs the AI server until shutdown.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	aicmd "github.com/louisbranch/fracturing.space/internal/cmd/ai"
)

func main() {
	cfg, err := aicmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[AI] ")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := aicmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
