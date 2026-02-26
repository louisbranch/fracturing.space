package webctx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

func TestWithResolvedUserIDReturnsBackgroundForNilRequest(t *testing.T) {
	t.Parallel()

	if got := WithResolvedUserID(nil, nil); got == nil {
		t.Fatalf("expected background context")
	}
}

func TestWithResolvedUserIDReturnsRequestContextWhenResolverMissing(t *testing.T) {
	t.Parallel()

	baseCtx := context.WithValue(context.Background(), struct{}{}, "ok")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(baseCtx)
	if got := WithResolvedUserID(req, nil); got != baseCtx {
		t.Fatalf("expected original request context")
	}
}

func TestWithResolvedUserIDReturnsRequestContextWhenResolverEmpty(t *testing.T) {
	t.Parallel()

	baseCtx := context.WithValue(context.Background(), struct{}{}, "ok")
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(baseCtx)
	if got := WithResolvedUserID(req, func(*http.Request) string { return "   " }); got != baseCtx {
		t.Fatalf("expected original request context")
	}
}

func TestWithResolvedUserIDInjectsOutgoingMetadata(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := WithResolvedUserID(req, func(*http.Request) string { return "user-123" })
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata")
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-123" {
		t.Fatalf("user id metadata = %v, want [user-123]", values)
	}
}
