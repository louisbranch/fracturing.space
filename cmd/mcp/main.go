package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	mcpcmd "github.com/louisbranch/fracturing.space/internal/cmd/mcp"
)

// main starts the MCP server on stdio or HTTP.
func main() {
	cfg, err := mcpcmd.ParseConfig(flag.CommandLine, os.Args[1:], func(key string) (string, bool) {
		value, ok := os.LookupEnv(key)
		return value, ok
	})
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}
	log.SetPrefix("[MCP] ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcpcmd.Run(ctx, cfg); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}
