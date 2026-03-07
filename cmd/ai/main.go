// Package main wires the AI gRPC service process lifecycle.
//
// It reads config from flags/env and runs the AI server until shutdown.
package main

import (
	"log"

	aicmd "github.com/louisbranch/fracturing.space/internal/cmd/ai"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[aicmd.Config]{
		Service:     platformcmd.ServiceAI,
		ParseConfig: aicmd.ParseConfig,
		Run:         aicmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
