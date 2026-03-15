package sessionctx

import (
	"context"
	"time"
)

// ToolInvocationContext carries the derived runtime context for one MCP call.
type ToolInvocationContext struct {
	RunCtx       context.Context
	Cancel       context.CancelFunc
	MCPContext   Context
	InvocationID string
}

// NewToolInvocationContext derives one tool-call execution context from the
// caller context and optional fixed MCP session authority.
func NewToolInvocationContext(ctx context.Context, getContext func() Context) (ToolInvocationContext, error) {
	return NewToolInvocationContextWithTimeout(ctx, getContext, CallTimeout)
}

// NewToolInvocationContextWithTimeout derives one tool-call execution context
// with an explicit timeout.
func NewToolInvocationContextWithTimeout(ctx context.Context, getContext func() Context, timeout time.Duration) (ToolInvocationContext, error) {
	mcpCtx := Context{}
	if getContext != nil {
		mcpCtx = getContext()
	}
	return NewToolInvocationContextWithContext(ctx, mcpCtx, timeout)
}

// NewToolInvocationContextWithContext derives one tool-call execution context
// from an already-resolved MCP session authority.
func NewToolInvocationContextWithContext(ctx context.Context, mcpCtx Context, timeout time.Duration) (ToolInvocationContext, error) {
	invocationID, err := newInvocationID()
	if err != nil {
		return ToolInvocationContext{}, err
	}

	runCtx, cancel := deriveToolRunContext(ctx, timeout)

	return ToolInvocationContext{
		RunCtx:       runCtx,
		Cancel:       cancel,
		MCPContext:   mcpCtx,
		InvocationID: invocationID,
	}, nil
}

func deriveToolRunContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, timeout)
}
