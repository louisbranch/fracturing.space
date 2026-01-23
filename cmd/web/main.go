// Package main implements a client for the Duality service.
package main

import (
	"context"
	"flag"
	"log"
	"time"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	addr = flag.String("addr", "localhost:8080", "the address to connect to")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDualityServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ActionRoll(ctx, &pb.ActionRollRequest{})
	if err != nil {
		log.Fatalf("could not perform action roll: %v", err)
	}
	log.Printf("Outcome: %s", r.GetOutcome())
}
