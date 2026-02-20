package grpc

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Dialer describes the gRPC dial behavior used by helpers.
type Dialer interface {
	DialContext(ctx context.Context, addr string, opts ...gogrpc.DialOption) (*gogrpc.ClientConn, error)
}

// DialerFunc adapts a dial function to the Dialer interface.
type DialerFunc func(ctx context.Context, addr string, opts ...gogrpc.DialOption) (*gogrpc.ClientConn, error)

// DialContext implements Dialer for DialerFunc.
func (fn DialerFunc) DialContext(ctx context.Context, addr string, opts ...gogrpc.DialOption) (*gogrpc.ClientConn, error) {
	return fn(ctx, addr, opts...)
}

// DialStage describes where a dial attempt failed.
type DialStage string

const (
	// DialStageConnect indicates a dial connection failure.
	DialStageConnect DialStage = "connect"
	// DialStageHealth indicates the health check failed.
	DialStageHealth DialStage = "health"
)

// DialError wraps dial and health check failures with a stage indicator.
type DialError struct {
	Stage DialStage
	Err   error
}

// Error implements the error interface.
func (e *DialError) Error() string {
	if e == nil {
		return "gRPC dial error"
	}
	return fmt.Sprintf("gRPC %s error: %v", e.Stage, e.Err)
}

// Unwrap returns the underlying error.
func (e *DialError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// DefaultClientDialOptions returns standard dial options for in-process clients.
// Includes OTel gRPC interceptors so that every outbound call propagates trace
// context automatically when a TracerProvider is registered.
func DefaultClientDialOptions() []gogrpc.DialOption {
	return []gogrpc.DialOption{
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
		gogrpc.WithBlock(),
		gogrpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

// DialWithHealth dials a gRPC endpoint and waits for the health check to serve.
// It closes the connection if the health check fails.
func DialWithHealth(ctx context.Context, dialer Dialer, addr string, dialTimeout time.Duration, logf func(string, ...any), opts ...gogrpc.DialOption) (*gogrpc.ClientConn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if dialer == nil {
		dialer = DialerFunc(gogrpc.DialContext)
	}

	dialCtx := ctx
	if dialTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, dialTimeout)
		defer cancel()
	}

	conn, err := dialer.DialContext(dialCtx, addr, opts...)
	if err != nil {
		return nil, &DialError{Stage: DialStageConnect, Err: err}
	}
	if err := WaitForHealth(dialCtx, conn, "", logf); err != nil {
		_ = conn.Close()
		return nil, &DialError{Stage: DialStageHealth, Err: err}
	}
	return conn, nil
}
