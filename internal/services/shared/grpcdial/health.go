package grpcdial

import (
	"context"
	"errors"
	"fmt"
	"time"

	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	gogrpc "google.golang.org/grpc"
)

// DialWithHealth dials a service endpoint and normalizes connect/health errors
// into stable, service-labeled messages for startup callers.
func DialWithHealth(
	ctx context.Context,
	addr string,
	timeout time.Duration,
	serviceLabel string,
	logf func(string, ...any),
	opts ...gogrpc.DialOption,
) (*gogrpc.ClientConn, error) {
	conn, err := platformgrpc.DialWithHealth(ctx, nil, addr, timeout, logf, opts...)
	if err != nil {
		return nil, NormalizeDialError(serviceLabel, addr, err)
	}
	return conn, nil
}

// NormalizeDialError maps platform DialError stages into stable startup error
// messages used by service-specific dial helpers.
func NormalizeDialError(serviceLabel, addr string, err error) error {
	var dialErr *platformgrpc.DialError
	if errors.As(err, &dialErr) {
		if dialErr.Stage == platformgrpc.DialStageHealth {
			return fmt.Errorf("%s gRPC health check failed for %s: %w", serviceLabel, addr, dialErr.Err)
		}
		return fmt.Errorf("dial %s gRPC %s: %w", serviceLabel, addr, dialErr.Err)
	}
	return fmt.Errorf("dial %s gRPC %s: %w", serviceLabel, addr, err)
}
