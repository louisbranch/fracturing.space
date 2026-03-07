// Package main starts the discovery gRPC service process lifecycle.
package main

import (
	"log"

	discoverycmd "github.com/louisbranch/fracturing.space/internal/cmd/discovery"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[discoverycmd.Config]{
		Service:     platformcmd.ServiceDiscovery,
		ParseConfig: discoverycmd.ParseConfig,
		Run:         discoverycmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
