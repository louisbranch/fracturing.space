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

// main starts the MCP server on stdio or HTTP.
func main() {
	addrFlag := flag.String("addr", "localhost:8080", "gRPC server address")
	httpAddrFlag := flag.String("http-addr", "localhost:8081", "HTTP server address (for HTTP transport)")
	transportFlag := flag.String("transport", "stdio", "Transport type: stdio or http")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.Run(ctx, *addrFlag, *httpAddrFlag, *transportFlag); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}
