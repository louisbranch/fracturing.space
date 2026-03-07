// Package main starts the chat real-time service and handles termination.
//
// The process is a transport adapter around chat room lifecycle and message
// streaming so campaign state remains owned by the game domain.
package main

import (
	"log"

	chatcmd "github.com/louisbranch/fracturing.space/internal/cmd/chat"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[chatcmd.Config]{
		Service:     platformcmd.ServiceChat,
		ParseConfig: chatcmd.ParseConfig,
		Run:         chatcmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
