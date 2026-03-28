// Package grpcassert provides shared gRPC assertion helpers for tests.
//
// These helpers centralise the assertStatusCode / assertStatusMessage
// pattern that was previously duplicated across 15+ transport test packages.
package grpcassert

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatusCode verifies that err is a gRPC status error with the given code.
// It fails the test immediately if err is nil, not a gRPC status, or carries
// a different code.
func StatusCode(t interface {
	Helper()
	Fatalf(format string, args ...any)
}, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v, got nil", want)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if st.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", st.Code(), want, st.Message())
	}
}

// StatusMessage verifies that err is a gRPC status error whose message
// contains the given substring.
func StatusMessage(t interface {
	Helper()
	Fatalf(format string, args ...any)
}, err error, substr string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substr)
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if !strings.Contains(st.Message(), substr) {
		t.Fatalf("status message %q does not contain %q", st.Message(), substr)
	}
}
