// Package main starts the social gRPC service process lifecycle.
package main

import (
	"log"

	socialcmd "github.com/louisbranch/fracturing.space/internal/cmd/social"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[socialcmd.Config]{
		Service:     platformcmd.ServiceSocial,
		ParseConfig: socialcmd.ParseConfig,
		Run:         socialcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
