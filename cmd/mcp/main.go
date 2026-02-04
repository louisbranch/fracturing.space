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
	addrDefault := getenvDefault("DUALITY_GRPC_ADDR", "localhost:8080")
	httpAddrDefault := getenvDefault("DUALITY_MCP_HTTP_ADDR", "localhost:8081")
	addrFlag := flag.String("addr", addrDefault, "gRPC server address")
	httpAddrFlag := flag.String("http-addr", httpAddrDefault, "HTTP server address (for HTTP transport)")
	transportFlag := flag.String("transport", "stdio", "Transport type: stdio or http")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.Run(ctx, *addrFlag, *httpAddrFlag, *transportFlag); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}

// getenvDefault returns the env value or a fallback when unset.
func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
