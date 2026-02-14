package grpc

import (
	"context"
	"fmt"
	"time"

	gogrpc "google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// WaitForHealth blocks until the gRPC health check reports SERVING or the context ends.
func WaitForHealth(ctx context.Context, conn *gogrpc.ClientConn, service string, logf func(string, ...any)) error {
	if conn == nil {
		return fmt.Errorf("gRPC connection is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	healthClient := grpc_health_v1.NewHealthClient(conn)
	backoff := 200 * time.Millisecond
	for {
		callCtx, cancel := context.WithTimeout(ctx, time.Second)
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

		if backoff < time.Second {
			backoff *= 2
			if backoff > time.Second {
				backoff = time.Second
			}
		}
	}
}
