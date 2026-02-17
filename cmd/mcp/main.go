// Package main starts the MCP server in stdio or HTTP transport mode.
//
// This keeps protocol transport selection in one place and shields tool behavior
// from deployment startup concerns.
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
	cfg, err := mcpcmd.ParseConfig(flag.CommandLine, os.Args[1:])
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
