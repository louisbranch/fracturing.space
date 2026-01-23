package main

import (
	"context"
	"flag"
	"log"

	"github.com/louisbranch/duality-protocol/internal/app/server"
)

var (
	port = flag.Int("port", 8080, "The server port")
)

func main() {
	flag.Parse()
	if err := server.Run(context.Background(), *port); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
