package requestctx

import (
	"context"
	"testing"
)

func TestUserIDFromContextRoundTrip(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-42")
	got := UserIDFromContext(ctx)
	if got != "user-42" {
		t.Fatalf("UserIDFromContext = %q, want %q", got, "user-42")
	}
}

func TestUserIDFromContextEmpty(t *testing.T) {
	got := UserIDFromContext(context.Background())
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestUserIDFromContextNil(t *testing.T) {
	got := UserIDFromContext(nil)
	if got != "" {
		t.Fatalf("expected empty string for nil context, got %q", got)
	}
}

func TestWithUserIDNilContext(t *testing.T) {
	ctx := WithUserID(nil, "user-99")
	if ctx == nil {
		t.Fatalf("expected non-nil context")
	}
	if got := UserIDFromContext(ctx); got != "user-99" {
		t.Fatalf("UserIDFromContext = %q, want %q", got, "user-99")
	}
}
