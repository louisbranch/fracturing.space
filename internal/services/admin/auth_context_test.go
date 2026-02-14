package admin

import (
	"context"
	"testing"
)

func TestAuthContextRoundTrip(t *testing.T) {
	ctx := contextWithAuthUser(context.Background(), "user-42")
	got := authUserFromContext(ctx)
	if got != "user-42" {
		t.Fatalf("authUserFromContext = %q, want %q", got, "user-42")
	}
}

func TestAuthContextEmpty(t *testing.T) {
	got := authUserFromContext(context.Background())
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestAuthContextNil(t *testing.T) {
	got := authUserFromContext(nil)
	if got != "" {
		t.Fatalf("expected empty string for nil context, got %q", got)
	}
}
