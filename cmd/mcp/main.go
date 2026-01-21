package main

import (
	"flag"
	"log"

	"github.com/louisbranch/duality-protocol/internal/mcp"
)

// main starts the MCP server on stdio.
func main() {
	addrFlag := flag.String("addr", "localhost:8080", "gRPC server address")
	flag.Parse()

	mcpServer, err := mcp.New(*addrFlag)
	if err != nil {
		log.Fatalf("failed to initialize MCP server: %v", err)
	}
	if err := mcpServer.Serve(); err != nil {
		log.Fatalf("failed to serve MCP: %v", err)
	}
}
