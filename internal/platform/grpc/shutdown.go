package grpc

import (
	"time"

	gogrpc "google.golang.org/grpc"
)

// DefaultGracefulStopTimeout is the maximum duration to wait for in-flight
// RPCs before forcing an immediate stop.
const DefaultGracefulStopTimeout = 10 * time.Second

// GracefulStopWithTimeout calls GracefulStop on the server but falls back to
// Stop if the graceful shutdown doesn't complete within the given timeout.
func GracefulStopWithTimeout(server *gogrpc.Server, timeout time.Duration) {
	if server == nil {
		return
	}
	if timeout <= 0 {
		timeout = DefaultGracefulStopTimeout
	}
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		server.Stop()
	}
}
