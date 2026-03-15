// Package sessionctx owns MCP session-scoped authority and request metadata
// plumbing.
//
// It intentionally sits outside tool/resource handler packages so contributors
// can distinguish bridge/runtime support code from gameplay-facing MCP domain
// handlers.
package sessionctx
