package validate

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RequiredID trims whitespace from raw and returns an InvalidArgument error
// if the result is empty. The name parameter is used in the error message
// (e.g. "campaign id").
func RequiredID(raw, name string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" {
		return "", status.Error(codes.InvalidArgument, name+" is required")
	}
	return id, nil
}
