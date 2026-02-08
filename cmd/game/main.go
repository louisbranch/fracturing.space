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
)

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, *port); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
