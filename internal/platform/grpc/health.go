package grpc

import (
	"context"
	"fmt"
	"time"

	gogrpc "google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

const (
	waitForHealthInitialBackoffDefault = 200 * time.Millisecond
	waitForHealthMaxBackoffDefault     = time.Second
	waitForHealthCallTimeoutDefault    = time.Second
)

// WaitForHealth blocks until the gRPC health check reports SERVING or the context ends.
func WaitForHealth(ctx context.Context, conn *gogrpc.ClientConn, service string, logf func(string, ...any)) error {
	return waitForHealthWithBackoff(
		ctx,
		conn,
		service,
		logf,
		waitForHealthInitialBackoffDefault,
		waitForHealthMaxBackoffDefault,
		waitForHealthCallTimeoutDefault,
	)
}

func waitForHealthWithBackoff(
	ctx context.Context,
	conn *gogrpc.ClientConn,
	service string,
	logf func(string, ...any),
	initialBackoff time.Duration,
	maxBackoff time.Duration,
	callTimeout time.Duration,
) error {
	if conn == nil {
		return fmt.Errorf("gRPC connection is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if initialBackoff <= 0 {
		initialBackoff = waitForHealthInitialBackoffDefault
	}
	if maxBackoff < initialBackoff {
		maxBackoff = initialBackoff
	}
	if callTimeout <= 0 {
		callTimeout = waitForHealthCallTimeoutDefault
	}

	healthClient := grpc_health_v1.NewHealthClient(conn)
	backoff := initialBackoff
	for {
		callCtx, cancel := context.WithTimeout(ctx, callTimeout)
		response, err := healthClient.Check(callCtx, &grpc_health_v1.HealthCheckRequest{Service: service})
		cancel()
		if err == nil && response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			if logf != nil {
				logf("gRPC health check is SERVING")
			}
			return nil
		}
		if logf != nil {
			if err != nil {
				logf("waiting for gRPC health: %v", err)
			} else {
				logf("waiting for gRPC health: status %s", response.GetStatus().String())
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for gRPC health: %w", ctx.Err())
		case <-time.After(backoff):
		}

		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}
