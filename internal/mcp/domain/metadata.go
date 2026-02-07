package domain

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/metadata"
)

// ToolCallMetadata carries correlation identifiers for MCP tool calls.
type ToolCallMetadata struct {
	RequestID    string
	InvocationID string
}

// ResourceUpdateNotifier notifies MCP clients about resource updates.
type ResourceUpdateNotifier func(ctx context.Context, uri string)

// NewInvocationID generates an invocation identifier for a tool call.
func NewInvocationID() (string, error) {
	return id.NewID()
}

// NewRequestID generates a request identifier for a gRPC call.
func NewRequestID() (string, error) {
	return id.NewID()
}

// NewOutgoingContext attaches request metadata to a context.
func NewOutgoingContext(ctx context.Context, invocationID string) (context.Context, ToolCallMetadata, error) {
	requestID, err := NewRequestID()
	if err != nil {
		return nil, ToolCallMetadata{}, err
	}

	callCtx := metadata.AppendToOutgoingContext(ctx, grpcmeta.RequestIDHeader, requestID)
	if invocationID != "" {
		callCtx = metadata.AppendToOutgoingContext(callCtx, grpcmeta.InvocationIDHeader, invocationID)
	}

	return callCtx, ToolCallMetadata{RequestID: requestID, InvocationID: invocationID}, nil
}

// MergeResponseMetadata overlays response headers on top of sent metadata.
func MergeResponseMetadata(sent ToolCallMetadata, header metadata.MD) ToolCallMetadata {
	requestID := grpcmeta.FirstMetadataValue(header, grpcmeta.RequestIDHeader)
	if requestID == "" {
		requestID = sent.RequestID
	}

	invocationID := grpcmeta.FirstMetadataValue(header, grpcmeta.InvocationIDHeader)
	if invocationID == "" {
		invocationID = sent.InvocationID
	}

	return ToolCallMetadata{RequestID: requestID, InvocationID: invocationID}
}

// CallToolResultWithMetadata builds a tool result with correlation metadata.
func CallToolResultWithMetadata(meta ToolCallMetadata) *mcp.CallToolResult {
	result := &mcp.CallToolResult{
		Meta: map[string]any{
			grpcmeta.RequestIDHeader: meta.RequestID,
		},
	}
	if meta.InvocationID != "" {
		result.Meta[grpcmeta.InvocationIDHeader] = meta.InvocationID
	}
	return result
}

// NotifyResourceUpdates sends resource update notifications for each URI provided.
func NotifyResourceUpdates(ctx context.Context, notify ResourceUpdateNotifier, uris ...string) {
	if notify == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	for _, uri := range uris {
		if strings.TrimSpace(uri) == "" {
			continue
		}
		notify(ctx, uri)
	}
}
