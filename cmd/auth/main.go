// Package main starts the auth gRPC boundary and exits cleanly on signal.
//
// The bootstrap path intentionally stays thin so auth configuration is isolated to
// flag/env parsing and server lifecycle management.
package main

import (
	"log"

	authcmd "github.com/louisbranch/fracturing.space/internal/cmd/auth"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[authcmd.Config]{
		Service:     platformcmd.ServiceAuth,
		ParseConfig: authcmd.ParseConfig,
		Run:         authcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
