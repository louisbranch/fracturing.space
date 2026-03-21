package app

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
)

func normalizeGRPCServeError(err error) error {
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve gRPC: %w", err)
}

// runGRPCServeLoop starts gRPC serving and waits for either context
// cancellation or server exit. On context cancellation, shutdownFn is called
// before waiting for serve to finish.
func runGRPCServeLoop(
	ctx context.Context,
	serveFn func() error,
	shutdownFn func(),
) error {
	if serveFn == nil {
		return fmt.Errorf("gRPC serve function is required")
	}
	if shutdownFn == nil {
		shutdownFn = func() {}
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- serveFn()
	}()

	select {
	case <-ctx.Done():
		shutdownFn()
		return normalizeGRPCServeError(<-serveErr)
	case err := <-serveErr:
		return normalizeGRPCServeError(err)
	}
}
