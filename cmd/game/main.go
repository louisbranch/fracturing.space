package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/services/game/app"
)

var (
	port = flag.Int("port", 8080, "The game server port")
	addr = flag.String("addr", "", "The game server listen address (overrides -port)")
)

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if *addr != "" {
		if err := server.RunWithAddr(ctx, *addr); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		return
	}
	if err := server.Run(ctx, *port); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
