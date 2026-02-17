// Package main hosts the admin control plane and terminates on standard service signals.
//
// It keeps startup focused on process lifecycle while delegating all admin domain
// coordination to internal/cmd/admin and internal/services/admin.
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

// main wires process configuration and hands execution to the admin command package.
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
