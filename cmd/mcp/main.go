// Package main starts the MCP server in stdio or HTTP transport mode.
//
// This keeps protocol transport selection in one place and shields tool behavior
// from deployment startup concerns.
package main

import (
	"log"

	mcpcmd "github.com/louisbranch/fracturing.space/internal/cmd/mcp"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

// main starts the MCP server on stdio or HTTP.
func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[mcpcmd.Config]{
		Service:     platformcmd.ServiceMCP,
		ParseConfig: mcpcmd.ParseConfig,
		Run:         mcpcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
