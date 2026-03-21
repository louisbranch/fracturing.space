// Package main starts the invite gRPC service process lifecycle.
package main

import (
	"log"

	invitecmd "github.com/louisbranch/fracturing.space/internal/cmd/invite"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[invitecmd.Config]{
		Service:     platformcmd.ServiceInvite,
		ParseConfig: invitecmd.ParseConfig,
		Run:         invitecmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
