package grpc

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/status"
	gogrpc "google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Backoff constants for managed connection health monitoring.
const (
	DefaultRetryDelay          = 500 * time.Millisecond
	MaxRetryDelay              = 10 * time.Second
	healthyPollIntervalDefault = 10 * time.Second
	healthCheckTimeout         = 2 * time.Second
)

// ManagedConnMode controls whether NewManagedConn blocks until the peer is
// healthy or returns immediately.
type ManagedConnMode int

const (
	// ModeOptional returns immediately. The connection is valid but RPCs
	// may fail with Unavailable until the peer is up.
	ModeOptional ManagedConnMode = iota
	// ModeRequired blocks until the first health check passes or the
	// context is cancelled.
	ModeRequired
)

// ManagedConnConfig configures a ManagedConn.
type ManagedConnConfig struct {
	// Name identifies this connection in logs (e.g., "game", "auth").
	Name string
	// Addr is the gRPC target address.
	Addr string
	// Mode controls blocking behavior on construction.
	Mode ManagedConnMode
	// DialOpts override the default LenientDialOptions().
	DialOpts []gogrpc.DialOption
	// Logf is the logging function. Defaults to log.Printf.
	Logf func(string, ...any)
	// StatusReporter receives health transitions for the capability.
	StatusReporter *status.Reporter
	// StatusCapability is the capability name reported to the status service.
	StatusCapability string

	// timing overrides are used by tests to keep feedback loops fast.
	retryDelay          time.Duration
	maxRetryDelay       time.Duration
	healthyPollInterval time.Duration
	checkTimeout        time.Duration

	// checkHealth overrides the default gRPC health probe. Used in tests.
	checkHealth func(ctx context.Context, conn *gogrpc.ClientConn) error
}

// ManagedConn wraps a gRPC client connection with background health monitoring
// and automatic status reporting. The underlying connection is created via
// non-blocking dial and is always non-nil after construction.
type ManagedConn struct {
	name string
	conn *gogrpc.ClientConn
	logf func(string, ...any)

	reporter   *status.Reporter
	capability string

	ready   chan struct{} // closed when first health check passes
	readyMu sync.Mutex    // protects readyOnce
	readyOk bool          // true after ready is closed

	cancel context.CancelFunc
	done   chan struct{} // closed when background goroutine exits

	retryDelay          time.Duration
	maxRetryDelay       time.Duration
	healthyPollInterval time.Duration

	// checkHealth is injectable for testing.
	checkHealth func(ctx context.Context, conn *gogrpc.ClientConn) error
}

// NewManagedConn creates a gRPC connection with background health monitoring.
//
// The connection is created via non-blocking dial (no WithBlock), so it is
// always non-nil unless dial options are invalid. gRPC handles transport-level
// reconnection automatically.
//
// For ModeRequired, this blocks until the first health check passes or the
// context is cancelled. For ModeOptional, it returns immediately.
func NewManagedConn(ctx context.Context, cfg ManagedConnConfig) (*ManagedConn, error) {
	if ctx == nil {
		return nil, fmt.Errorf("managed conn %s: context is required", cfg.Name)
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("managed conn %s: address is required", cfg.Name)
	}

	logf := cfg.Logf
	if logf == nil {
		logf = log.Printf
	}
	dialOpts := cfg.DialOpts
	if len(dialOpts) == 0 {
		dialOpts = LenientDialOptions()
	}

	conn, err := gogrpc.NewClient(cfg.Addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("managed conn %s: dial: %w", cfg.Name, err)
	}

	check := cfg.checkHealth
	if check == nil {
		check = newDefaultCheckHealth(cfg.checkTimeout)
	}
	retryDelay := cfg.retryDelay
	if retryDelay <= 0 {
		retryDelay = DefaultRetryDelay
	}
	maxRetryDelay := cfg.maxRetryDelay
	if maxRetryDelay <= 0 {
		maxRetryDelay = MaxRetryDelay
	}
	healthyPollInterval := cfg.healthyPollInterval
	if healthyPollInterval <= 0 {
		healthyPollInterval = healthyPollIntervalDefault
	}

	monitorCtx, cancel := context.WithCancel(context.Background())
	mc := &ManagedConn{
		name:                cfg.Name,
		conn:                conn,
		logf:                logf,
		reporter:            cfg.StatusReporter,
		capability:          cfg.StatusCapability,
		ready:               make(chan struct{}),
		cancel:              cancel,
		done:                make(chan struct{}),
		retryDelay:          retryDelay,
		maxRetryDelay:       maxRetryDelay,
		healthyPollInterval: healthyPollInterval,
		checkHealth:         check,
	}

	go mc.monitor(monitorCtx)

	if cfg.Mode == ModeRequired {
		if err := mc.WaitReady(ctx); err != nil {
			mc.Close()
			return nil, fmt.Errorf("managed conn %s: wait ready: %w", cfg.Name, err)
		}
	}

	return mc, nil
}

// Conn returns the underlying gRPC connection. It is always non-nil after
// successful construction.
func (mc *ManagedConn) Conn() *gogrpc.ClientConn {
	return mc.conn
}

// Ready reports whether the peer has passed at least one health check.
func (mc *ManagedConn) Ready() bool {
	select {
	case <-mc.ready:
		return true
	default:
		return false
	}
}

// WaitReady blocks until the peer passes a health check or the context ends.
func (mc *ManagedConn) WaitReady(ctx context.Context) error {
	select {
	case <-mc.ready:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("managed conn %s: %w", mc.name, ctx.Err())
	}
}

// Close stops the background health monitor and closes the underlying
// gRPC connection.
func (mc *ManagedConn) Close() error {
	mc.cancel()
	<-mc.done
	return mc.conn.Close()
}

func (mc *ManagedConn) monitor(ctx context.Context) {
	defer close(mc.done)

	wasHealthy := false
	backoff := mc.retryDelay

	for {
		if ctx.Err() != nil {
			return
		}

		err := mc.checkHealth(ctx, mc.conn)
		healthy := err == nil

		switch {
		case healthy && !wasHealthy:
			mc.logf("managed conn %s: healthy", mc.name)
			mc.markReady()
			if mc.reporter != nil && mc.capability != "" {
				mc.reporter.SetOperational(mc.capability)
			}
			wasHealthy = true
			backoff = mc.retryDelay

		case !healthy && wasHealthy:
			mc.logf("managed conn %s: unhealthy: %v", mc.name, err)
			if mc.reporter != nil && mc.capability != "" {
				mc.reporter.SetUnavailable(mc.capability, err.Error())
			}
			wasHealthy = false
		}

		// Choose sleep interval based on current health state.
		interval := backoff
		if wasHealthy {
			interval = mc.healthyPollInterval
		} else if !healthy {
			// Increase backoff for next unhealthy cycle.
			if backoff < mc.maxRetryDelay {
				backoff *= 2
				if backoff > mc.maxRetryDelay {
					backoff = mc.maxRetryDelay
				}
			}
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (mc *ManagedConn) markReady() {
	mc.readyMu.Lock()
	defer mc.readyMu.Unlock()
	if !mc.readyOk {
		mc.readyOk = true
		close(mc.ready)
	}
}

// newDefaultCheckHealth performs a gRPC health check with a short timeout.
func newDefaultCheckHealth(timeout time.Duration) func(context.Context, *gogrpc.ClientConn) error {
	if timeout <= 0 {
		timeout = healthCheckTimeout
	}

	return func(ctx context.Context, conn *gogrpc.ClientConn) error {
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		client := grpc_health_v1.NewHealthClient(conn)
		resp, err := client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{Service: ""})
		if err != nil {
			return err
		}
		if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
			return fmt.Errorf("health status: %s", resp.GetStatus().String())
		}
		return nil
	}
}
