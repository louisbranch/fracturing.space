// Package main starts the browser-facing play service.
package main

import (
	"log"

	playcmd "github.com/louisbranch/fracturing.space/internal/cmd/play"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[playcmd.Config]{
		Service:     platformcmd.ServicePlay,
		ParseConfig: playcmd.ParseConfig,
		Run:         playcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
