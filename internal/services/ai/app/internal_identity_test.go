package server

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestValidateIncomingServiceIdentity(t *testing.T) {
	validate := validateIncomingServiceIdentity(map[string]struct{}{
		"ai":     {},
		"worker": {},
	})

	t.Run("allows calls without service identity", func(t *testing.T) {
		if err := validate(context.Background()); err != nil {
			t.Fatalf("validate() error = %v", err)
		}
	})

	t.Run("allows configured service identity", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))
		if err := validate(ctx); err != nil {
			t.Fatalf("validate() error = %v", err)
		}
	})

	t.Run("rejects unknown service identity", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "web"))
		err := validate(ctx)
		if status.Code(err) != codes.PermissionDenied {
			t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.PermissionDenied, err)
		}
	})
}
