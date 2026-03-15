// Package main starts the internal MCP HTTP bridge.
//
// This keeps MCP startup wiring in one place and shields tool behavior from
// deployment concerns.
package main

import (
	"log"

	mcpcmd "github.com/louisbranch/fracturing.space/internal/cmd/mcp"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

// main starts the MCP server.
func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[mcpcmd.Config]{
		Service:     platformcmd.ServiceMCP,
		ParseConfig: mcpcmd.ParseConfig,
		Run:         mcpcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
