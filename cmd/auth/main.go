package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	authcmd "github.com/louisbranch/fracturing.space/internal/cmd/auth"
)

func main() {
	cfg, err := authcmd.ParseConfig(flag.CommandLine, os.Args[1:], func(key string) (string, bool) {
		value, ok := os.LookupEnv(key)
		return value, ok
	})
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[AUTH] ")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := authcmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
