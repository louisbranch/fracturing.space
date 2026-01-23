package server

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestServeStopsOnContext verifies the server serves and stops on cancel.
func TestServeStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	addr := grpcServer.listener.Addr().String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split address %q: %v", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	addr = net.JoinHostPort(host, port)

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	client := pb.NewDualityServiceClient(conn)
	callCtx, callCancel := context.WithTimeout(context.Background(), time.Second)
	defer callCancel()
	if _, err := client.ActionRoll(callCtx, &pb.ActionRollRequest{}); err != nil {
		t.Fatalf("action roll: %v", err)
	}

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop in time")
	}
}

// TestRunPortInUse verifies Run returns an error when the port is occupied.
func TestRunPortInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split address %q: %v", listener.Addr().String(), err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port %q: %v", port, err)
	}

	if err := Run(context.Background(), portNumber); err == nil {
		t.Fatal("expected error when port is already in use")
	}
}

// TestServeReturnsOnCancel verifies Serve returns promptly on cancel without connections.
func TestServeReturnsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(ctx)
	}()

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop on cancel")
	}
}

// TestServeReturnsErrorOnClosedListener verifies Serve reports listener errors.
func TestServeReturnsErrorOnClosedListener(t *testing.T) {
	grpcServer, err := New(0)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	if err := grpcServer.listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := grpcServer.Serve(ctx); err == nil {
		t.Fatal("expected serve error after closing listener")
	}
}
