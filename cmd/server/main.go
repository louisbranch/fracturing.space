package main

import (
	"flag"
	"log"

	"github.com/louisbranch/duality-protocol/internal/app/server"
)

var (
	port = flag.Int("port", 8080, "The server port")
)

func main() {
	flag.Parse()
	grpcServer, err := server.New(*port)
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}
	if err := grpcServer.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
