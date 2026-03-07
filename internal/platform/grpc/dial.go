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

// LenientDialOptions returns dial options without WithBlock, suitable for
// non-blocking connection attempts where the caller tolerates initial unavailability.
func LenientDialOptions() []gogrpc.DialOption {
	return []gogrpc.DialOption{
		gogrpc.WithTransportCredentials(insecure.NewCredentials()),
		gogrpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}
}

// DialLenient attempts to connect to a gRPC endpoint with a short timeout.
// On failure it returns nil conn and logs a warning instead of returning an error.
// Callers must handle a nil connection gracefully.
func DialLenient(ctx context.Context, addr string, logf func(string, ...any), opts ...gogrpc.DialOption) *gogrpc.ClientConn {
	if ctx == nil {
		ctx = context.Background()
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if len(opts) == 0 {
		opts = LenientDialOptions()
	}

	dialCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := gogrpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		logf("lenient dial %s failed: %v", addr, err)
		return nil
	}

	// Quick health check — don't block if unavailable.
	if err := WaitForHealth(dialCtx, conn, "", logf); err != nil {
		logf("lenient dial %s: health check failed, connection may be usable later: %v", addr, err)
		// Return the connection anyway — gRPC reconnects automatically.
		return conn
	}
	return conn
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
