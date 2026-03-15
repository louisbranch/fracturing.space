package sessionctx

import (
	"context"
	"testing"
	"time"
)

func TestDeriveToolRunContext(t *testing.T) {
	t.Run("applies timeout when caller has no deadline", func(t *testing.T) {
		ctx, cancel := deriveToolRunContext(context.Background(), 50*time.Millisecond)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected derived deadline")
		}
		if time.Until(deadline) > 100*time.Millisecond {
			t.Fatalf("expected derived deadline near timeout, got %v", deadline)
		}
	})

	t.Run("preserves caller deadline when present", func(t *testing.T) {
		parent, parentCancel := context.WithTimeout(context.Background(), time.Second)
		defer parentCancel()

		ctx, cancel := deriveToolRunContext(parent, 50*time.Millisecond)
		defer cancel()

		parentDeadline, ok := parent.Deadline()
		if !ok {
			t.Fatal("expected parent deadline")
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected inherited deadline")
		}
		if !deadline.Equal(parentDeadline) {
			t.Fatalf("expected inherited deadline %v, got %v", parentDeadline, deadline)
		}
	})

	t.Run("handles nil context", func(t *testing.T) {
		ctx, cancel := deriveToolRunContext(nil, 50*time.Millisecond)
		defer cancel()

		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("expected deadline on nil-derived context")
		}
	})
}

func TestNewToolInvocationContext(t *testing.T) {
	t.Run("uses getter context and default timeout", func(t *testing.T) {
		inv, err := NewToolInvocationContext(context.Background(), func() Context {
			return Context{CampaignID: "camp-1", SessionID: "sess-1", ParticipantID: "part-1"}
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer inv.Cancel()

		if inv.InvocationID == "" {
			t.Fatal("expected invocation ID")
		}
		if inv.MCPContext.CampaignID != "camp-1" || inv.MCPContext.SessionID != "sess-1" || inv.MCPContext.ParticipantID != "part-1" {
			t.Fatalf("unexpected MCP context: %#v", inv.MCPContext)
		}
		if _, ok := inv.RunCtx.Deadline(); !ok {
			t.Fatal("expected default timeout deadline")
		}
	})
}

func TestNewToolInvocationContextWithTimeout(t *testing.T) {
	t.Run("handles nil getter and zero timeout", func(t *testing.T) {
		inv, err := NewToolInvocationContextWithTimeout(context.Background(), nil, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer inv.Cancel()

		if inv.InvocationID == "" {
			t.Fatal("expected invocation ID")
		}
		if inv.MCPContext != (Context{}) {
			t.Fatalf("expected empty MCP context, got %#v", inv.MCPContext)
		}
		if _, ok := inv.RunCtx.Deadline(); ok {
			t.Fatal("expected no deadline for zero timeout")
		}
	})
}

func TestNewToolInvocationContextWithContext(t *testing.T) {
	t.Run("preserves explicit MCP context", func(t *testing.T) {
		parent, parentCancel := context.WithTimeout(context.Background(), time.Second)
		defer parentCancel()

		inv, err := NewToolInvocationContextWithContext(parent, Context{CampaignID: "camp-2"}, 50*time.Millisecond)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer inv.Cancel()

		if inv.InvocationID == "" {
			t.Fatal("expected invocation ID")
		}
		if inv.MCPContext.CampaignID != "camp-2" {
			t.Fatalf("campaign_id = %q, want %q", inv.MCPContext.CampaignID, "camp-2")
		}
		if got, ok := inv.RunCtx.Deadline(); !ok {
			t.Fatal("expected inherited deadline")
		} else if want, _ := parent.Deadline(); !got.Equal(want) {
			t.Fatalf("deadline = %v, want %v", got, want)
		}
	})
}
