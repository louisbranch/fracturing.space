package domain

import "github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"

// These aliases keep MCP handler signatures readable while session-scoped
// bridge authority and request metadata live in the dedicated sessionctx
// support package.
type Context = sessionctx.Context
type ResourceUpdateNotifier = sessionctx.ResourceUpdateNotifier
type ToolCallMetadata = sessionctx.ToolCallMetadata
