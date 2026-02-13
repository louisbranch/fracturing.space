// Package main hosts the admin dashboard service.
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

	"github.com/louisbranch/fracturing.space/internal/services/admin"
)

var (
	// defaultHTTPAddr sets the fallback HTTP listen address.
	defaultHTTPAddr = ":8082"
	// defaultGRPCAddr sets the fallback game server address.
	defaultGRPCAddr = "localhost:8080"
	// defaultAuthAddr sets the fallback auth server address.
	defaultAuthAddr = "localhost:8083"
)

// envOrDefault returns the trimmed environment value or a fallback.
func envOrDefault(keys []string, fallback string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return fallback
}

// main runs the web server with optional gRPC connectivity.
func main() {
	httpAddr := flag.String("http-addr", envOrDefault([]string{"FRACTURING_SPACE_ADMIN_ADDR"}, defaultHTTPAddr), "HTTP listen address")
	grpcAddr := flag.String("grpc-addr", envOrDefault([]string{"FRACTURING_SPACE_GAME_ADDR"}, defaultGRPCAddr), "game server address")
	authAddr := flag.String("auth-addr", envOrDefault([]string{"FRACTURING_SPACE_AUTH_ADDR"}, defaultAuthAddr), "auth server address")
	flag.Parse()
	log.SetPrefix("[ADMIN] ")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server, err := admin.NewServer(ctx, admin.Config{
		HTTPAddr:        *httpAddr,
		GRPCAddr:        *grpcAddr,
		AuthAddr:        *authAddr,
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
