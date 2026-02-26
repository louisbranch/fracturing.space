package web

import (
	"errors"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
)

func TestGRPCErrorHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		fallback int
		want     int
	}{
		{
			name:     "invalid argument",
			err:      status.Error(codes.InvalidArgument, "invalid"),
			fallback: http.StatusBadGateway,
			want:     http.StatusBadRequest,
		},
		{
			name:     "unauthenticated",
			err:      status.Error(codes.Unauthenticated, "unauthenticated"),
			fallback: http.StatusBadGateway,
			want:     http.StatusUnauthorized,
		},
		{
			name:     "permission denied",
			err:      status.Error(codes.PermissionDenied, "forbidden"),
			fallback: http.StatusBadGateway,
			want:     http.StatusForbidden,
		},
		{
			name:     "not found",
			err:      status.Error(codes.NotFound, "not found"),
			fallback: http.StatusBadGateway,
			want:     http.StatusNotFound,
		},
		{
			name:     "failed precondition",
			err:      status.Error(codes.FailedPrecondition, "failed precondition"),
			fallback: http.StatusBadGateway,
			want:     http.StatusConflict,
		},
		{
			name:     "unavailable",
			err:      status.Error(codes.Unavailable, "unavailable"),
			fallback: http.StatusBadGateway,
			want:     http.StatusServiceUnavailable,
		},
		{
			name:     "unmapped grpc code falls back",
			err:      status.Error(codes.Internal, "internal"),
			fallback: http.StatusBadGateway,
			want:     http.StatusBadGateway,
		},
		{
			name:     "non grpc error falls back",
			err:      errors.New("boom"),
			fallback: http.StatusBadGateway,
			want:     http.StatusBadGateway,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := websupport.GRPCErrorHTTPStatus(tc.err, tc.fallback)
			if got != tc.want {
				t.Fatalf("status = %d, want %d", got, tc.want)
			}
		})
	}
}
