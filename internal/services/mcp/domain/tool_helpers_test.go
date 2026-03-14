package domain

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
