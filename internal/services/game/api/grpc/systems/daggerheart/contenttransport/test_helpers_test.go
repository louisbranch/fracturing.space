package contenttransport

import (
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()
	got := status.Code(err)
	if got != want {
		t.Fatalf("status code = %v, want %v (err=%v)", got, want, err)
	}
}
