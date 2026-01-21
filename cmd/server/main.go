package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	port = flag.Int("port", 8080, "The server port")
)

type server struct {
	pb.UnimplementedDiceRollServiceServer
}

func (s *server) ActionRoll(ctx context.Context, in *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ActionRoll is not implemented yet")
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDiceRollServiceServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
