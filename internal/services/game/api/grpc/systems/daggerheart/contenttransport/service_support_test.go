package contenttransport

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestContentStoreMissing(t *testing.T) {
	var nilHandler *Handler
	_, err := nilHandler.contentStore()
	grpcassert.StatusCode(t, err, codes.Internal)

	handler := &Handler{}
	_, err = handler.contentStore()
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestMapContentErr(t *testing.T) {
	err := mapContentErr("get class", storage.ErrNotFound)
	grpcassert.StatusCode(t, err, codes.NotFound)

	err = mapContentErr("get class", errors.New("boom"))
	grpcassert.StatusCode(t, err, codes.Internal)
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if statusErr.Message() != "get class" {
		t.Fatalf("message = %q, want %q", statusErr.Message(), "get class")
	}
}
