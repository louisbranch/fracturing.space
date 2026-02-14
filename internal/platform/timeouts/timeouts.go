// Package timeouts defines shared timeout constants used across services.
// Centralizing these values prevents drift between service boundaries and
// makes the durations discoverable.
package timeouts

import "time"

// GRPCDial caps the wait time when dialing a gRPC peer.
const GRPCDial = 2 * time.Second

// GRPCRequest caps the time allowed for a single gRPC request from the
// admin dashboard to a backend service.
const GRPCRequest = 2 * time.Second

// ReadHeader limits how long an HTTP server waits for request headers.
const ReadHeader = 5 * time.Second

// Shutdown limits how long an HTTP server waits for in-flight requests
// during graceful shutdown.
const Shutdown = 5 * time.Second
