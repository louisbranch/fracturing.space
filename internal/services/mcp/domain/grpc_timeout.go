package domain

import "time"

// grpcCallTimeout caps the time for a single gRPC call from an MCP tool handler.
const grpcCallTimeout = 5 * time.Second

// grpcLongCallTimeout caps the time for gRPC calls that involve heavier operations
// such as fork creation or full event replay.
const grpcLongCallTimeout = 10 * time.Second
