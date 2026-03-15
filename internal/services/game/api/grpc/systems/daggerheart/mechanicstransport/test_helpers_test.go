package mechanicstransport

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestHandler(seed int64) *Handler {
	return NewHandler(func() (int64, error) { return seed, nil })
}

func intPointer(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func stringPointer(value string) *string {
	return &value
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("status code = %v, want %v (err=%v)", got, want, err)
	}
}
