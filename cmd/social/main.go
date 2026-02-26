// Package main starts the social gRPC service process lifecycle.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	socialcmd "github.com/louisbranch/fracturing.space/internal/cmd/social"
)

func main() {
	cfg, err := socialcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[SOCIAL] ")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := socialcmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
