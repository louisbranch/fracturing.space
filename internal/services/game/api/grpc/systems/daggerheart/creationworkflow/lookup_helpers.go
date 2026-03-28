package creationworkflow

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func invalidContentLookup(ctx context.Context, err error, internalMessage, invalidMessage string, args ...any) error {
	if err == nil {
		return nil
	}
	if grpcerror.OptionalLookupErrorContext(ctx, err, internalMessage) == nil {
		return status.Errorf(codes.InvalidArgument, invalidMessage, args...)
	}
	return grpcerror.LookupErrorContext(ctx, err, internalMessage, "")
}
