package eventtransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}

func assertStatusMessage(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q", substr)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if msg := statusErr.Message(); !containsSubstring(msg, substr) {
		t.Fatalf("status message = %q, want to contain %q", msg, substr)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
