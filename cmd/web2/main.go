// Package main starts the clean-slate web2 service.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	web2cmd "github.com/louisbranch/fracturing.space/internal/cmd/web2"
)

func main() {
	cfg, err := web2cmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[WEB2] ")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := web2cmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
