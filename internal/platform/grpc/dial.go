package grpc

import (
	"context"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	gogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultLenientDialTimeout = 2 * time.Second

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
	return DialLenientWithTimeout(ctx, addr, defaultLenientDialTimeout, logf, opts...)
}

// DialLenientWithTimeout attempts to connect to a gRPC endpoint with caller-defined timeout.
// On failure it returns nil conn and logs a warning instead of returning an error.
// Callers must handle a nil connection gracefully.
func DialLenientWithTimeout(ctx context.Context, addr string, dialTimeout time.Duration, logf func(string, ...any), opts ...gogrpc.DialOption) *gogrpc.ClientConn {
	if ctx == nil {
		ctx = context.Background()
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	if len(opts) == 0 {
		opts = LenientDialOptions()
	}
	if dialTimeout <= 0 {
		dialTimeout = defaultLenientDialTimeout
	}

	dialCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, err := gogrpc.NewClient(addr, opts...)
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
