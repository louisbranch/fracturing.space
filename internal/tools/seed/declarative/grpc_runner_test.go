package declarative

import (
	"context"
	"errors"
	"testing"

	seed "github.com/louisbranch/fracturing.space/internal/tools/seed"
)

func TestResolveLocalFallbackAddr_PreservesResolvedHost(t *testing.T) {
	t.Parallel()

	originalLookup := seed.LookupHost
	t.Cleanup(func() {
		seed.LookupHost = originalLookup
	})
	seed.LookupHost = func(context.Context, string) ([]string, error) {
		return []string{"10.0.0.1"}, nil
	}

	got := seed.ResolveLocalFallbackAddr("game:8082")
	if got != "game:8082" {
		t.Fatalf("seed.ResolveLocalFallbackAddr() = %q, want %q", got, "game:8082")
	}
}

func TestResolveLocalFallbackAddr_FallsBackToLoopback(t *testing.T) {
	t.Parallel()

	originalLookup := seed.LookupHost
	t.Cleanup(func() {
		seed.LookupHost = originalLookup
	})
	seed.LookupHost = func(context.Context, string) ([]string, error) {
		return nil, errors.New("lookup failed")
	}

	got := seed.ResolveLocalFallbackAddr("game:8082")
	if got != "127.0.0.1:8082" {
		t.Fatalf("seed.ResolveLocalFallbackAddr() = %q, want %q", got, "127.0.0.1:8082")
	}
}
