package ai

import (
	"github.com/louisbranch/fracturing.space/internal/services/ai/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// serviceErrorToStatus maps a service-layer error to a gRPC status error.
func serviceErrorToStatus(err error) error {
	if err == nil {
		return nil
	}
	kind := service.ErrorKindOf(err)
	return status.Error(kindToGRPC(kind), err.Error())
}

func kindToGRPC(kind service.ErrorKind) codes.Code {
	switch kind {
	case service.ErrKindInvalidArgument:
		return codes.InvalidArgument
	case service.ErrKindNotFound:
		return codes.NotFound
	case service.ErrKindAlreadyExists:
		return codes.AlreadyExists
	case service.ErrKindPermissionDenied:
		return codes.PermissionDenied
	case service.ErrKindFailedPrecondition:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}
