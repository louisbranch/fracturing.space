package validate_test

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequiredID(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		field   string
		want    string
		wantErr codes.Code
	}{
		{name: "valid", raw: "abc-123", field: "campaign id", want: "abc-123"},
		{name: "trims spaces", raw: "  abc  ", field: "campaign id", want: "abc"},
		{name: "trims tabs", raw: "\tabc\t", field: "campaign id", want: "abc"},
		{name: "empty", raw: "", field: "campaign id", wantErr: codes.InvalidArgument},
		{name: "whitespace only", raw: "   ", field: "scene id", wantErr: codes.InvalidArgument},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validate.RequiredID(tt.raw, tt.field)
			if tt.wantErr != 0 {
				if err == nil {
					t.Fatalf("expected error with code %v, got nil", tt.wantErr)
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %T", err)
				}
				if st.Code() != tt.wantErr {
					t.Fatalf("code = %v, want %v", st.Code(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
