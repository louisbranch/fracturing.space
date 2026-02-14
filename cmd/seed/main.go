// Package main provides a CLI for seeding the local development database
// with demo data by exercising the full MCP→game stack, or by generating
// dynamic scenarios directly via gRPC.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	seedcmd "github.com/louisbranch/fracturing.space/internal/cmd/seed"
)

func main() {
	cfg, err := seedcmd.ParseConfig(flag.CommandLine, os.Args[1:], func(key string) (string, bool) {
		value, ok := os.LookupEnv(key)
		return value, ok
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if err := seedcmd.Run(ctx, cfg, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
