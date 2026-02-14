// Package main hosts the admin dashboard service.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	admincmd "github.com/louisbranch/fracturing.space/internal/cmd/admin"
)

// main runs the web server with optional gRPC connectivity.
func main() {
	cfg, err := admincmd.ParseConfig(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[ADMIN] ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := admincmd.Run(ctx, cfg); err != nil {
		log.Fatalf("serve web: %v", err)
	}
}
