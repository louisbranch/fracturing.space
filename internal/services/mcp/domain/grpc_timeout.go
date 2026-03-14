package domain

import "time"

// grpcCallTimeout caps the time for a single gRPC call from an MCP tool handler
// when the surrounding MCP transport does not provide a caller deadline.
//
// The timeout must stay comfortably above normal cold-start and bootstrap paths
// so MCP does not undercut the caller-visible transport contract.
const grpcCallTimeout = 30 * time.Second

// grpcLongCallTimeout caps the time for gRPC calls that involve heavier operations
// such as fork creation or full event replay.
const grpcLongCallTimeout = 60 * time.Second
