package outcometransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestHandlerRequireSessionOutcomeDependencies(t *testing.T) {
	handler := &Handler{}
	err := handler.requireSessionOutcomeDependencies()
	grpcassert.StatusCode(t, err, codes.Internal)

	handler, _, _ = newTestHandler()
	if err := handler.requireSessionOutcomeDependencies(); err != nil {
		t.Fatalf("requireSessionOutcomeDependencies returned error: %v", err)
	}
}

func TestHandlerRequireRollOutcomeDependencies(t *testing.T) {
	handler := &Handler{}
	err := handler.requireRollOutcomeDependencies()
	grpcassert.StatusCode(t, err, codes.Internal)

	handler, _, _ = newTestHandler()
	if err := handler.requireRollOutcomeDependencies(); err != nil {
		t.Fatalf("requireRollOutcomeDependencies returned error: %v", err)
	}
}

func TestClamp(t *testing.T) {
	if got := clamp(-1, 0, 5); got != 0 {
		t.Fatalf("clamp low = %d, want 0", got)
	}
	if got := clamp(6, 0, 5); got != 5 {
		t.Fatalf("clamp high = %d, want 5", got)
	}
	if got := clamp(3, 0, 5); got != 3 {
		t.Fatalf("clamp middle = %d, want 3", got)
	}
}
