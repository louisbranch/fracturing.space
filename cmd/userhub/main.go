// Package main starts the userhub gRPC service process lifecycle.
package main

import (
	"log"

	userhubcmd "github.com/louisbranch/fracturing.space/internal/cmd/userhub"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[userhubcmd.Config]{
		Service:     platformcmd.ServiceUserHub,
		ParseConfig: userhubcmd.ParseConfig,
		Run:         userhubcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
