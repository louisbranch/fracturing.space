package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisbranch/duality-engine/internal/app/mcp"
)

// main starts the MCP server on stdio.
func main() {
	addrFlag := flag.String("addr", "localhost:8080", "gRPC server address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.Run(ctx, *addrFlag); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}
