package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/services/auth/app"
)

var (
	port     = flag.Int("port", 8083, "The auth gRPC server port")
	httpAddr = flag.String("http-addr", envOrDefault([]string{"FRACTURING_SPACE_AUTH_HTTP_ADDR"}, "localhost:8084"), "The auth HTTP server address")
)

func main() {
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, *port, *httpAddr); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func envOrDefault(keys []string, fallback string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return fallback
}
