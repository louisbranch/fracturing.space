package declarative

import (
	"context"
	"errors"
	"testing"
)

func TestResolveLocalFallbackAddr_PreservesResolvedHost(t *testing.T) {
	t.Parallel()

	originalLookup := seedLookupHost
	t.Cleanup(func() {
		seedLookupHost = originalLookup
	})
	seedLookupHost = func(context.Context, string) ([]string, error) {
		return []string{"10.0.0.1"}, nil
	}

	got := resolveLocalFallbackAddr("game:8082")
	if got != "game:8082" {
		t.Fatalf("resolveLocalFallbackAddr() = %q, want %q", got, "game:8082")
	}
}

func TestResolveLocalFallbackAddr_FallsBackToLoopback(t *testing.T) {
	t.Parallel()

	originalLookup := seedLookupHost
	t.Cleanup(func() {
		seedLookupHost = originalLookup
	})
	seedLookupHost = func(context.Context, string) ([]string, error) {
		return nil, errors.New("lookup failed")
	}

	got := resolveLocalFallbackAddr("game:8082")
	if got != "127.0.0.1:8082" {
		t.Fatalf("resolveLocalFallbackAddr() = %q, want %q", got, "127.0.0.1:8082")
	}
}
