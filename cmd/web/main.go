// Package main hosts the Duality web client.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/louisbranch/duality-engine/internal/web"
)

var (
	// defaultHTTPAddr sets the fallback HTTP listen address.
	defaultHTTPAddr = ":8082"
	// defaultGRPCAddr sets the fallback gRPC server address.
	defaultGRPCAddr = "localhost:8080"
)

// envOrDefault returns the trimmed environment value or a fallback.
func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// main runs the web server with optional gRPC connectivity.
func main() {
	httpAddr := flag.String("http-addr", envOrDefault("DUALITY_WEB_ADDR", defaultHTTPAddr), "HTTP listen address")
	grpcAddr := flag.String("grpc-addr", envOrDefault("DUALITY_GRPC_ADDR", defaultGRPCAddr), "gRPC server address")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := web.NewServer(ctx, web.Config{
		HTTPAddr:        *httpAddr,
		GRPCAddr:        *grpcAddr,
		GRPCDialTimeout: 2 * time.Second,
	})
	if err != nil {
		log.Fatalf("init web server: %v", err)
	}
	defer server.Close()

	if err = server.ListenAndServe(ctx); err != nil {
		log.Fatalf("serve web: %v", err)
	}
}
