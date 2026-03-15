package adversarytransport

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func statusError(code codes.Code, message string) error {
	return status.Error(code, message)
}
