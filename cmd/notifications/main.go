// Package main starts the notifications gRPC service process lifecycle.
package main

import (
	"log"

	notificationscmd "github.com/louisbranch/fracturing.space/internal/cmd/notifications"
	platformcmd "github.com/louisbranch/fracturing.space/internal/platform/cmd"
)

func main() {
	if err := platformcmd.RunServiceMain(platformcmd.ServiceMainOptions[notificationscmd.Config]{
		Service:     platformcmd.ServiceNotifications,
		ParseConfig: notificationscmd.ParseConfig,
		Run:         notificationscmd.Run,
	}); err != nil {
		log.Fatal(err)
	}
}
