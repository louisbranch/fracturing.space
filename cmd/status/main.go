// Package main starts the status gRPC service and owns its process lifecycle.
package main

import (
	"log"

	statuscmd "github.com/louisbranch/fracturing.space/internal/cmd/status"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[statuscmd.Config]{
		Service:     platformcmd.ServiceStatus,
		ParseConfig: statuscmd.ParseConfig,
		Run:         statuscmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
