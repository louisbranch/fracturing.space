// Package main starts the browser-facing web service.
//
// This process owns route wiring and static template serving so campaign/auth
// context is translated consistently for browsers and web users.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	webcmd "github.com/louisbranch/fracturing.space/internal/cmd/web"
)

func main() {
	cfg, err := webcmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[WEB] ")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := webcmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
