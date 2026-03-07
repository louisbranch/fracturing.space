// Package main starts the worker service process lifecycle.
package main

import (
	"log"

	workercmd "github.com/louisbranch/fracturing.space/internal/cmd/worker"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[workercmd.Config]{
		Service:     platformcmd.ServiceWorker,
		ParseConfig: workercmd.ParseConfig,
		Run:         workercmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
