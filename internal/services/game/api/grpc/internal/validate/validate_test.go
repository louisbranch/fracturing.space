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

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		field   string
		maxLen  int
		wantErr bool
	}{
		{name: "within limit", value: "hello", field: "name", maxLen: 10},
		{name: "exact limit", value: "hello", field: "name", maxLen: 5},
		{name: "exceeds limit", value: "hello!", field: "name", maxLen: 5, wantErr: true},
		{name: "empty passes", value: "", field: "name", maxLen: 5},
		{name: "zero limit blocks non-empty", value: "a", field: "name", maxLen: 0, wantErr: true},
		{name: "zero limit allows empty", value: "", field: "name", maxLen: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate.MaxLength(tt.value, tt.field, tt.maxLen)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				st, ok := status.FromError(err)
				if !ok {
					t.Fatalf("expected gRPC status error, got %T", err)
				}
				if st.Code() != codes.InvalidArgument {
					t.Fatalf("code = %v, want InvalidArgument", st.Code())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
