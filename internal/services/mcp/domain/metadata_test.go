package domain

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	grpcmetadata "google.golang.org/grpc/metadata"
)

func TestNewOutgoingContext(t *testing.T) {
	t.Run("attaches request ID without admin override", func(t *testing.T) {
		ctx, meta, err := NewOutgoingContext(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.RequestID == "" {
			t.Fatal("expected non-empty request ID")
		}

		md, ok := grpcmetadata.FromOutgoingContext(ctx)
		if !ok {
			t.Fatal("expected outgoing metadata")
		}
		if got := md.Get(grpcmeta.RequestIDHeader); len(got) == 0 || got[0] != meta.RequestID {
			t.Errorf("expected request ID %q in metadata, got %v", meta.RequestID, got)
		}
		// Admin override is handled by connection-level interceptors, not per-call.
		if got := md.Get(grpcmeta.PlatformRoleHeader); len(got) != 0 {
			t.Errorf("expected no platform role in per-call metadata, got %v", got)
		}
		if got := md.Get(grpcmeta.AuthzOverrideReasonHeader); len(got) != 0 {
			t.Errorf("expected no override reason in per-call metadata, got %v", got)
		}
	})

	t.Run("attaches invocation ID when provided", func(t *testing.T) {
		ctx, meta, err := NewOutgoingContext(context.Background(), "inv-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.InvocationID != "inv-123" {
			t.Errorf("expected invocation ID %q, got %q", "inv-123", meta.InvocationID)
		}

		md, _ := grpcmetadata.FromOutgoingContext(ctx)
		if got := md.Get(grpcmeta.InvocationIDHeader); len(got) == 0 || got[0] != "inv-123" {
			t.Errorf("expected invocation ID in metadata, got %v", got)
		}
	})

	t.Run("omits invocation ID when empty", func(t *testing.T) {
		ctx, _, err := NewOutgoingContext(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		md, _ := grpcmetadata.FromOutgoingContext(ctx)
		if got := md.Get(grpcmeta.InvocationIDHeader); len(got) != 0 {
			t.Errorf("expected no invocation ID in metadata, got %v", got)
		}
	})
}

func TestNewOutgoingContextWithContext(t *testing.T) {
	t.Run("propagates participant ID", func(t *testing.T) {
		mcpCtx := Context{ParticipantID: "part-456"}
		ctx, _, err := NewOutgoingContextWithContext(context.Background(), "inv-1", mcpCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		md, _ := grpcmetadata.FromOutgoingContext(ctx)
		if got := md.Get(grpcmeta.ParticipantIDHeader); len(got) == 0 || got[0] != "part-456" {
			t.Errorf("expected participant ID in metadata, got %v", got)
		}
	})

	t.Run("skips whitespace-only participant ID", func(t *testing.T) {
		mcpCtx := Context{ParticipantID: "  "}
		ctx, _, err := NewOutgoingContextWithContext(context.Background(), "", mcpCtx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		md, _ := grpcmetadata.FromOutgoingContext(ctx)
		if got := md.Get(grpcmeta.ParticipantIDHeader); len(got) != 0 {
			t.Errorf("expected no participant ID in metadata, got %v", got)
		}
	})
}

func TestMergeResponseMetadata(t *testing.T) {
	t.Run("overrides with response values", func(t *testing.T) {
		sent := ToolCallMetadata{RequestID: "sent-req", InvocationID: "sent-inv"}
		header := grpcmetadata.Pairs(
			grpcmeta.RequestIDHeader, "resp-req",
			grpcmeta.InvocationIDHeader, "resp-inv",
		)

		merged := MergeResponseMetadata(sent, header)
		if merged.RequestID != "resp-req" {
			t.Errorf("expected request ID %q, got %q", "resp-req", merged.RequestID)
		}
		if merged.InvocationID != "resp-inv" {
			t.Errorf("expected invocation ID %q, got %q", "resp-inv", merged.InvocationID)
		}
	})

	t.Run("falls back to sent values when response empty", func(t *testing.T) {
		sent := ToolCallMetadata{RequestID: "sent-req", InvocationID: "sent-inv"}
		merged := MergeResponseMetadata(sent, grpcmetadata.MD{})
		if merged.RequestID != "sent-req" {
			t.Errorf("expected request ID %q, got %q", "sent-req", merged.RequestID)
		}
		if merged.InvocationID != "sent-inv" {
			t.Errorf("expected invocation ID %q, got %q", "sent-inv", merged.InvocationID)
		}
	})

	t.Run("falls back to sent values when header is nil", func(t *testing.T) {
		sent := ToolCallMetadata{RequestID: "r1", InvocationID: "i1"}
		merged := MergeResponseMetadata(sent, nil)
		if merged.RequestID != "r1" {
			t.Errorf("expected request ID %q, got %q", "r1", merged.RequestID)
		}
	})
}

func TestCallToolResultWithMetadata(t *testing.T) {
	t.Run("includes request ID", func(t *testing.T) {
		meta := ToolCallMetadata{RequestID: "req-1"}
		result := CallToolResultWithMetadata(meta)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Meta[grpcmeta.RequestIDHeader] != "req-1" {
			t.Errorf("expected request ID in meta, got %v", result.Meta)
		}
	})

	t.Run("includes invocation ID when present", func(t *testing.T) {
		meta := ToolCallMetadata{RequestID: "req-1", InvocationID: "inv-1"}
		result := CallToolResultWithMetadata(meta)
		if result.Meta[grpcmeta.InvocationIDHeader] != "inv-1" {
			t.Errorf("expected invocation ID in meta, got %v", result.Meta)
		}
	})

	t.Run("omits invocation ID when empty", func(t *testing.T) {
		meta := ToolCallMetadata{RequestID: "req-1"}
		result := CallToolResultWithMetadata(meta)
		if _, ok := result.Meta[grpcmeta.InvocationIDHeader]; ok {
			t.Error("expected no invocation ID in meta")
		}
	})
}

func TestNotifyResourceUpdates(t *testing.T) {
	t.Run("nil notifier does not panic", func(t *testing.T) {
		NotifyResourceUpdates(context.Background(), nil, "uri://1")
	})

	t.Run("nil context uses background", func(t *testing.T) {
		var called []string
		notifier := func(_ context.Context, uri string) {
			called = append(called, uri)
		}
		NotifyResourceUpdates(nil, notifier, "uri://1")
		if len(called) != 1 || called[0] != "uri://1" {
			t.Errorf("expected one call with uri://1, got %v", called)
		}
	})

	t.Run("skips empty URIs", func(t *testing.T) {
		var called []string
		notifier := func(_ context.Context, uri string) {
			called = append(called, uri)
		}
		NotifyResourceUpdates(context.Background(), notifier, "", "  ", "uri://valid")
		if len(called) != 1 || called[0] != "uri://valid" {
			t.Errorf("expected one call with uri://valid, got %v", called)
		}
	})

	t.Run("notifies all valid URIs", func(t *testing.T) {
		var called []string
		notifier := func(_ context.Context, uri string) {
			called = append(called, uri)
		}
		NotifyResourceUpdates(context.Background(), notifier, "uri://1", "uri://2")
		if len(called) != 2 {
			t.Errorf("expected 2 calls, got %d", len(called))
		}
	})
}
