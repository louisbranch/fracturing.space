package sessionctx

import "time"

const (
	// CallTimeout caps the time for a single gRPC call from an MCP tool handler
	// or resource read.
	CallTimeout = 30 * time.Second

	// LongCallTimeout caps gRPC calls that perform heavier mutation or history
	// work but should still fail deterministically.
	LongCallTimeout = 60 * time.Second
)
