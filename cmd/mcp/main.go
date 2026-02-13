package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	mcp "github.com/louisbranch/fracturing.space/internal/services/mcp/app"
)

// main starts the MCP server on stdio or HTTP.
func main() {
	addrDefault := getenvDefault([]string{"FRACTURING_SPACE_GAME_ADDR"}, "localhost:8080")
	httpAddrDefault := getenvDefault([]string{"FRACTURING_SPACE_MCP_HTTP_ADDR"}, "localhost:8081")
	addrFlag := flag.String("addr", addrDefault, "game server address")
	httpAddrFlag := flag.String("http-addr", httpAddrDefault, "HTTP server address (for HTTP transport)")
	transportFlag := flag.String("transport", "stdio", "Transport type: stdio or http")
	flag.Parse()
	log.SetPrefix("[MCP] ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := mcp.Run(ctx, *addrFlag, *httpAddrFlag, *transportFlag); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}

// getenvDefault returns the env value or a fallback when unset.
func getenvDefault(keys []string, fallback string) string {
	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			return value
		}
	}
	return fallback
}
